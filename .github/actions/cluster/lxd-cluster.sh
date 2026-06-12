#!/bin/bash

set -e

#================================================
# Variables
#================================================

# Cluster name and size.
CLUSTER_NAME="${CLUSTER_NAME:-cls}"
CLUSTER_SIZE="${CLUSTER_SIZE:-3}"

# Image to use for cluster instances.
INSTANCE_IMAGE="${INSTANCE_IMAGE:-ubuntu:24.04}"

# Type of cluster instances (container or virtual-machine).
INSTANCE_TYPE="${INSTANCE_TYPE:-container}"

# Component versions.
VERSION_LXD="${VERSION_LXD:-latest/edge}"
VERSION_MICROOVN="${VERSION_MICROOVN:-latest/edge}"
VERSION_MICROCEPH="${VERSION_MICROCEPH:-latest/edge}"

# MicroOVN configuration.
MICROOVN_ENABLED="${MICROOVN_ENABLED:-false}"
MICROOVN_PKI_DIR="/var/snap/microovn/common/data/pki"
MICROCEPH_ENABLED="${MICROCEPH_ENABLED:-false}"
MICROCEPH_INSTANCE="${CLUSTER_NAME}-ceph"

# MinIO configuration.
MINIO_ENABLED="${MINIO_ENABLED:-true}"
MINIO_INSTALL_DIR="${MINIO_PATH:-/usr/local/bin}"

# Other.
INSTANCE="${CLUSTER_NAME}"
LEADER="${CLUSTER_NAME}-1"
STORAGE_POOL="${CLUSTER_NAME}-pool"
STORAGE_DRIVER="dir"
NETWORK_NAME="${CLUSTER_NAME}br0"

#================================================
# Utils
#================================================

# waitInstance waits for the VM to become ready.
waitInstance() {
        local instance="$1"
        local timeout="${2:-60}"

        if [ "${instance}" = "" ]; then
                echo "Error: waitInstance: missing argument: instance name"
                return 1
        fi

        echo "Waiting instance ${instance} to become ready ..."
        for j in $(seq 1 "${timeout}"); do
                local procCount=$(lxc info "${instance}" | awk '/Processes:/ {print $2}')
                if [ "${procCount:-0}" -gt 0 ]; then
                        echo "Instance ${instance} ready after ${j} seconds."
                        break
                fi

                if [ "${j}" -ge "${timeout}" ]; then
                        echo "Error: Instance ${instance} still not ready after ${timeout} seconds!"
                        return 1
                fi

                sleep 1
        done
}

# instanceIPv4 returns the IPv4 address of the instance with the given name.
instanceIPv4() {
        instance="$1"

        # Try for enp5s0 (VM) and eth0 (container) interfaces.
        for inf in enp5s0 eth0; do
                ipv4=$(lxc ls "${instance}" -f csv -c 4 | grep -oP "(\d{1,3}\.){3}\d{1,3}(?= \(${inf}\))" || true)
                if [ "${ipv4}" != "" ]; then
                        echo "${ipv4}"
                        return
                fi
        done

        echo "Error: Failed to obtain IPv4 address of instance ${instance}"
        return 1
}

#========================
# Cluster setup
#========================

# deploy deploys instances required for a LXD cluster.
deploy() {
        if [ "${MICROCEPH_ENABLED}" = "true" ] && [ "${INSTANCE_TYPE}" != "virtual-machine" ]; then
                echo "Error: MicroCeph setup requires virtual-machine cluster members."
                exit 1
        fi

        # Create dedicated network.
        echo "Creating network ${NETWORK_NAME} ..."
        exists=$(lxc network list --format csv | awk -F, '{print $1}' | grep "${NETWORK_NAME}" || true)
        if [ ! "${exists}" ]; then
                lxc network create "${NETWORK_NAME}"
        fi

        # Create storage pool.
        echo "Creating storage pool ${STORAGE_POOL} ..."
        exists=$(lxc storage list --format csv | awk -F, '{print $1}' | grep "${STORAGE_POOL}" || true)
        if [ ! "${exists}" ]; then
                lxc storage create "${STORAGE_POOL}" zfs
        fi

        # Setup cluster VMs.
        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                instance="${INSTANCE}-${i}"

                state=$(lxc list --format csv --columns s "${instance}")
                case "${state}" in
                "RUNNING")
                        echo "Instance ${instance} already running."
                        continue
                        ;;
                "STOPPED")
                        echo "Starting instance ${instance}..."
                        lxc start "${instance}"
                        continue
                        ;;
                esac

                args=""
                if [ "${INSTANCE_TYPE}" = "virtual-machine" ]; then
                        args="--vm"
                else
                        args="-c security.nesting=true"
                fi

                echo "Creating instance ${instance} ..."

                lxc launch "${INSTANCE_IMAGE}" "${instance}" \
                        --storage "${STORAGE_POOL}" \
                        --network "${NETWORK_NAME}" \
                        -c limits.cpu=4 \
                        -c limits.memory=4GiB \
                        $args
        done

        # Setup dedicated MicroCeph instance.
        if [ "${MICROCEPH_ENABLED}" = "true" ]; then
                instance="${MICROCEPH_INSTANCE}"

                state=$(lxc list --format csv --columns s "${instance}")
                case "${state}" in
                "RUNNING")
                        echo "Instance ${instance} already running."
                        ;;
                "STOPPED")
                        echo "Starting instance ${instance}..."
                        lxc start "${instance}"
                        ;;
                *)
                        echo "Creating instance ${instance} ..."
                        lxc launch "${INSTANCE_IMAGE}" "${instance}" \
                                --storage "${STORAGE_POOL}" \
                                --network "${NETWORK_NAME}" \
                                -c limits.cpu=2 \
                                -c limits.memory=2GiB \
                                --vm
                        ;;
                esac
        fi

        # Wait for instances to become ready.
        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                instance="${INSTANCE}-${i}"
                waitInstance "${instance}"
                lxc exec "${instance}" -- systemctl is-system-running --wait
        done

        if [ "${MICROCEPH_ENABLED}" = "true" ]; then
                waitInstance "${MICROCEPH_INSTANCE}"
                lxc exec "${MICROCEPH_INSTANCE}" -- systemctl is-system-running --wait
        fi

        # Install LXD on VMs.
        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                instance="${INSTANCE}-${i}"

                echo "Preparing instance ${instance} ..."

                # Install snap daemon.
                lxc exec "${instance}" --env=DEBIAN_FRONTEND=noninteractive -- apt-get update
                lxc exec "${instance}" --env=DEBIAN_FRONTEND=noninteractive -- apt-get -qq -y install snapd

                # Install LXD snap.
                lxc exec "${instance}" -- snap refresh lxd --channel "${VERSION_LXD}" --cohort=+ || lxc exec "${instance}" -- snap install lxd --channel "${VERSION_LXD}" --cohort=+
        done

        echo "Cluster instances created."
        lxc list
}

# configure_lxd configures LXD cluster.
configure_lxd() {
        echo "Creating LXD cluster ..."

        # Create LXD cluster.
        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                instance="${INSTANCE}-${i}"

                isClustered=$(lxc exec "${instance}" -- lxc cluster list 2> /dev/null || true)
                if [ "${isClustered}" ]; then
                        continue
                fi

                # Get IPv4 of the instance.
                ipv4=$(instanceIPv4 "${instance}")

                # On the leader instance, just enable clustering and continue.
                if [ "${instance}" = "${LEADER}" ]; then
                        lxc exec "${instance}" -- lxc config set core.https_address "${ipv4}"
                        lxc exec "${instance}" -- lxc cluster enable "${instance}"
                        continue
                fi

                # Create and extract token for a new cluster member.
                token=$(lxc exec "${LEADER}" -- lxc cluster add -q "${instance}")
                if [ "${token}" = "" ]; then
                        echo "Error: Failed retrieveing join token for instance ${instance}"
                        exit 1
                fi

                # Apply the cluster member configuration.
                lxc exec "${instance}" -- lxd init --preseed << EOF
cluster:
  enabled: true
  server_address: ${ipv4}
  cluster_token: ${token}
EOF
        done

        # Install and configure MinIO on each cluster member.
        if [ "${MINIO_ENABLED}" == "true" ]; then
                curl -sSfL https://dl.min.io/server/minio/release/linux-amd64/minio --output "/tmp/minio"
                curl -sSfL https://dl.min.io/client/mc/release/linux-amd64/mc --output "/tmp/mc"

                chmod +x "/tmp/minio"
                chmod +x "/tmp/mc"

                for i in $(seq 1 "${CLUSTER_SIZE}"); do
                        instance="${INSTANCE}-${i}"
                        hasBucketSupport=$(lxc exec "${instance}" -- lxc info | grep -e "- storage_buckets" || true)

                        # Install MinIO if enabled.
                        if [ "${hasBucketSupport}" != "" ]; then
                                echo "Installing MinIO server and client on instance ${instance} ..."

                                lxc exec "${instance}" -- mkdir -p "${MINIO_INSTALL_DIR}"

                                # Upload MinIO sever and client binaries.
                                lxc file push --quiet /tmp/minio "${instance}/${MINIO_INSTALL_DIR}/minio"
                                lxc file push --quiet /tmp/mc "${instance}${MINIO_INSTALL_DIR}/mc"

                                # Configure MinIO.
                                lxc exec "${instance}" -- snap set lxd minio.path="${MINIO_INSTALL_DIR}"
                                lxc exec "${instance}" -- snap restart lxd
                                lxc exec "${instance}" -- lxd waitready --timeout 30
                                lxc exec "${instance}" -- lxc config set core.storage_buckets_address ":8555" || true
                        fi
                done

                rm /tmp/minio /tmp/mc
        fi

        # Create default storage pool.
        exists=$(lxc exec "${LEADER}" -- lxc storage list | grep "default" || true)
        if [ ! "${exists}" ]; then
                for i in $(seq 1 "${CLUSTER_SIZE}"); do
                        instance="${INSTANCE}-${i}"
                        lxc exec "${LEADER}" -- lxc storage create default "${STORAGE_DRIVER}" --target "${instance}"
                done

                lxc exec "${LEADER}" -- lxc storage create default "${STORAGE_DRIVER}"
                lxc exec "${LEADER}" -- lxc profile device add default root disk pool=default path=/

                # Resize default storage.
                if [ "${STORAGE_DRIVER}" != "dir" ]; then
                        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                                instance="${INSTANCE}-${i}"
                                lxc exec "${LEADER}" -- lxc storage set default size 3GiB --target "${instance}"
                        done
                fi
        fi

        # Create default managed network (lxdbr0).
        exists=$(lxc exec "${LEADER}" -- lxc network list | grep "lxdbr0" || true)
        if [ ! "${exists}" ]; then
                for i in $(seq 1 "${CLUSTER_SIZE}"); do
                        instance="${INSTANCE}-${i}"
                        lxc exec "${LEADER}" -- lxc network create lxdbr0 --target "${instance}"
                done

                lxc exec "${LEADER}" -- lxc network create lxdbr0
                lxc exec "${LEADER}" -- lxc profile device add default eth0 nic nictype=bridged parent=lxdbr0
        fi

        configure_ovn
        configure_microceph

        # Configure new cluster remote.
        token=$(lxc exec "${LEADER}" -- lxc config trust add --name host --quiet)
        ipv4=$(instanceIPv4 "${LEADER}")

        lxc remote rm "${CLUSTER_NAME}" 2>/dev/null || true
        lxc remote add "${CLUSTER_NAME}" "${ipv4}" --token "${token}"
        lxc remote switch "${CLUSTER_NAME}"

        # Show final cluster.
        lxc list
        lxc cluster list "${CLUSTER_NAME}:"
}

# configure_ovn installs and configures MicroOVN on LXD cluster members.
configure_ovn() {
        if [ "${MICROOVN_ENABLED}" != "true" ]; then
                echo "OVN setup disabled."
                return
        fi

        if [ "${INSTANCE_TYPE}" != "virtual-machine" ]; then
                echo "Error: OVN setup requires virtual-machine cluster members."
                exit 1
        fi

        echo "Installing MicroOVN on cluster members ..."

        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                instance="${INSTANCE}-${i}"
                lxc exec "${instance}" -- snap install microovn --channel "${VERSION_MICROOVN}"

                if lxc exec "${instance}" -- snap connections lxd | grep -qwF lxd:ovn-certificates; then
                        lxc exec "${instance}" -- snap connect lxd:ovn-certificates microovn:ovn-certificates
                        lxc exec "${instance}" -- snap connect lxd:ovn-chassis microovn:ovn-chassis
                fi
        done

        echo "Forming MicroOVN cluster ..."

        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                instance="${INSTANCE}-${i}"

                # On the leader instance, bootstrap a new MicroOVN cluster and continue.
                if [ "${instance}" = "${LEADER}" ]; then
                        lxc exec "${instance}" -- microovn cluster bootstrap
                        continue
                fi

                # Create and extract a join token for a new cluster member.
                token=$(lxc exec "${LEADER}" -- microovn cluster add "${instance}")
                if [ "${token}" = "" ]; then
                        echo "Error: Failed retrieving MicroOVN join token for instance ${instance}"
                        exit 1
                fi

                lxc exec "${instance}" -- microovn cluster join "${token}"
        done

        # Wait for MicroOVN to become ready on all members.
        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                instance="${INSTANCE}-${i}"
                lxc exec "${instance}" -- microovn waitready
        done

        if ! info=$(lxc exec "${LEADER}" -- lxc info); then
                echo "Failed to get LXD info from ${LEADER}"
                exit 1
        fi

        if ! echo "${info}" | grep -qxF -- "- ovn_dynamic_northbound_connection"; then
                # LXD 5.0 resolves OVN southbound info through host ovs-vsctl and uses legacy OVN
                # client cert paths.
                for i in $(seq 1 "${CLUSTER_SIZE}"); do
                        instance="${INSTANCE}-${i}"

                        lxc exec "${instance}" -- mkdir -p /var/run/openvswitch /etc/ovn
                        lxc exec "${instance}" -- ln -sf /var/snap/microovn/common/run/switch/db.sock /var/run/openvswitch/db.sock
                        lxc exec "${instance}" -- ln -sf "${MICROOVN_PKI_DIR}/client-cert.pem" /etc/ovn/cert_host
                        lxc exec "${instance}" -- ln -sf "${MICROOVN_PKI_DIR}/client-privkey.pem" /etc/ovn/key_host
                        lxc exec "${instance}" -- ln -sf "${MICROOVN_PKI_DIR}/cacert.pem" /etc/ovn/ovn-central.crt
                done

                leaderIPv4=$(instanceIPv4 "${LEADER}")
                lxc exec "${LEADER}" -- lxc config set network.ovn.northbound_connection="ssl:${leaderIPv4}:6641"
        fi
}

# configure_microceph installs and bootstraps MicroCeph on a dedicated instance, then distributes
# its client configuration and keyring to every cluster member so they can use Ceph-backed storage pools.
configure_microceph() {
        if [ "${MICROCEPH_ENABLED}" != "true" ]; then
                echo "MicroCeph setup disabled."
                return
        fi

        microceph="${MICROCEPH_INSTANCE}"

        echo "Installing and configuring MicroCeph on ${microceph} ..."
        lxc exec "${microceph}" -- snap install microceph --channel="${VERSION_MICROCEPH}"

        # Create a loop device backed by a sparse file to use as the OSD disk.
        loopDevice=$(lxc exec "${microceph}" -- sh -c '
                truncate -s 10G /root/microceph.img
                losetup --show -f /root/microceph.img
        ')

        if [ -z "${loopDevice}" ]; then
                echo "Error: Failed to create loop device for MicroCeph OSD disk"
                return 1
        fi

        lxc exec "${microceph}" -- microceph cluster bootstrap
        lxc exec "${microceph}" -- microceph.ceph config set global mon_allow_pool_size_one true
        lxc exec "${microceph}" -- microceph.ceph config set global mon_allow_pool_delete true
        lxc exec "${microceph}" -- microceph.ceph config set global osd_pool_default_size 1
        lxc exec "${microceph}" -- microceph.ceph config set global osd_memory_target 939524096 # 896MiB = 768MiB (osd_memory_base) + 128MiB (osd_memory_cache_min)
        lxc exec "${microceph}" -- microceph.ceph osd crush rule rm replicated_rule
        lxc exec "${microceph}" -- microceph.ceph osd crush rule create-replicated replicated default osd

        for flag in nosnaptrim nobackfill norebalance norecover noscrub nodeep-scrub; do
                lxc exec "${microceph}" -- microceph.ceph osd set "${flag}"
        done

        lxc exec "${microceph}" -- microceph disk add --wipe "${loopDevice}"

        # Expose the MicroCeph configuration at the standard Ceph client path,
        # which is where the LXD snap's ceph storage driver expects to find it.
        lxc exec "${microceph}" -- sh -c "rm -rf /etc/ceph && ln -s /var/snap/microceph/current/conf /etc/ceph"

        lxc exec "${microceph}" -- microceph enable rgw
        lxc exec "${microceph}" -- microceph.ceph osd pool create cephfs_meta 32
        lxc exec "${microceph}" -- microceph.ceph osd pool create cephfs_data 32
        lxc exec "${microceph}" -- microceph.ceph fs new cephfs cephfs_meta cephfs_data

        lxc exec "${microceph}" --env=DEBIAN_FRONTEND=noninteractive -- apt-get update
        lxc exec "${microceph}" --env=DEBIAN_FRONTEND=noninteractive -- apt-get --no-install-recommends -qq -y install ceph-common

        echo "Waiting for MicroCeph on ${microceph} to become ready ..."
        for j in $(seq 1 60); do
                if ! pgStat=$(lxc exec "${microceph}" -- microceph.ceph pg stat 2>/dev/null); then
                        pgStat=""
                fi

                if [ -n "${pgStat}" ] && ! echo "${pgStat}" | grep -wq unknown; then
                        echo "MicroCeph on ${microceph} ready after ${j} seconds."
                        break
                fi

                if [ "${j}" -ge 60 ]; then
                        echo "Error: MicroCeph on ${microceph} still has unknown placement groups after 60 seconds!"
                        lxc exec "${microceph}" -- microceph.ceph status || true
                        return 1
                fi

                sleep 1
        done

        lxc exec "${microceph}" -- microceph.ceph status

        # Distribute the Ceph client configuration and keyring to every cluster member.
        cephDir=$(mktemp -d)
        lxc file pull "${microceph}/etc/ceph/ceph.conf" "${cephDir}/ceph.conf"
        lxc file pull "${microceph}/etc/ceph/ceph.client.admin.keyring" "${cephDir}/ceph.client.admin.keyring"

        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                instance="${INSTANCE}-${i}"

                lxc exec "${instance}" --env=DEBIAN_FRONTEND=noninteractive -- apt-get -qq -y install ceph-common
                lxc exec "${instance}" -- mkdir -p /etc/ceph
                lxc file push --quiet "${cephDir}/ceph.conf" "${instance}/etc/ceph/ceph.conf"
                lxc file push --quiet "${cephDir}/ceph.client.admin.keyring" "${instance}/etc/ceph/ceph.client.admin.keyring"
        done

        rm -rf "${cephDir}"

        # Export environment variables consumed by acceptance test pre-checks.
        cephIPv4=$(instanceIPv4 "${MICROCEPH_INSTANCE}")
        {
                echo "LXD_CEPH_CLUSTER=ceph"
                echo "LXD_CEPH_CEPHFS=cephfs"
                echo "LXD_CEPH_CEPHOBJECT_RADOSGW=http://${cephIPv4}"
        } >> "${GITHUB_ENV}"
}

#================================================
# Cleanup
#================================================

# cleanup removes the deployed resources.
#
cleanup() {
        # Remove VMs.
        echo "Removing instances ..."
        for i in $(seq 1 "${CLUSTER_SIZE}"); do
                instance="${INSTANCE}-${i}"
                lxc delete "${instance}" --force || true
        done

        if [ "${MICROCEPH_ENABLED}" = "true" ]; then
                lxc delete "${MICROCEPH_INSTANCE}" --force || true
        fi

        # Remove storage pool.
        echo "Removing storage pool ${STORAGE_POOL} ..."
        lxc storage delete "${STORAGE_POOL}" || true

        # Remove network.
        echo "Removing network ${NETWORK_NAME} ..."
        lxc network delete "${NETWORK_NAME}"  || true

        # Remove remote.
        lxc remote switch local
        lxc remote rm "${CLUSTER_NAME}" 2>/dev/null || true
}

#================================================
# Script
#================================================

action="${1:-}"
case "${action}" in
        deploy)
                echo "==> RUN: Deploy"

                deploy
                configure_lxd

                echo ""
                echo "==> DONE: LXD cluster created"
                ;;
        cleanup)
                echo "==> RUN: Cleanup"
                cleanup

                echo ""
                echo "==> Done: LXD cluster removed"
                ;;
        *)
                echo "Unkown action: ${action}"
                echo "Valid actions are: [deploy, cleanup]"
                echo "Run: $0 <action>"
                exit 1
                ;;
esac
