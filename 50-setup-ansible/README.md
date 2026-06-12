## 仮想マシン起動後に自動的に Ansible プレイブックを適用する

仮想サーバーの起動後に、マニフェストに指定したAnsibleのプレイブックを使って、
自動設定することができます。ただし、そのためにいくつかの条件があります。

- spec.auth.url で公開鍵を指定
- spec.auth.user で root を指定
- spec.ansible-playbook で プレイブックのパスを設定
- spec.networkInterface.[0].networkname で host-birdge を設定
- spec.networkInterface.[0].address で sshできる IPアドレスを設定

## サンプルマニフェスト

このYAMLで仮想サーバーを自動設定できます。

```yaml
apiVersion: v1
kind: Server
metadata:
    name: server-41
    comment: Ansibleプレイブックを自動実行するサーバー
spec:
    cpu: 1
    memory: 1024
    osVariant: ubuntu24.04
    # VMホストからAnsible を実行するために、rootユーザーに公開鍵をセット
    auth:
        url: https://github.com/takara9.keys
        user: root
    # Ansible プレイブックのパスをセット
    ansible-playbook:  playbook/setup.yaml
    # VMホストからssh可能なように、host-bridgeを使用して、IPアドレスをセット
    networkInterface:
        - networkname: host-bridge
          address: 192.168.1.41
          netmasklen: 24
          nameservers:
            addresses:
                - 192.168.1.9
            search:
                - labo.local
          routes:
            - to: default
              via: 192.168.1.1
```

## Ansibleプレイブックの書き方

hostsに、all を設定します。これ以外を設定するとプレイブックが適用されなくなります。

```yaml
- name: Setup the guest OS
  gather_facts: yes
  vars:
  hosts: all
  become: yes
  roles:
  tasks:
  - debug: msg="{{ ansible_facts }}"
  - name: パッケージのインストール
    apt:
      name: "{{ packages }}"
      state: present
    vars:
      packages:
        - gcc
        - make
```

## 仮想サーバーの起動と設定方法

実際にサーバーを起動して、Ansible Playbookを適用した結果は以下です。

```console
ubuntu@ws1:~/marmot-manifests/09-setup-ansible$ mactl create -f server-41.yaml 
リソースの作成要求が受け入れられました。ID: 5da0d
OS起動待機中.....
playbook 適用開始.....

PLAY [Setup the guest OS] ********************************************************************************************************************************************

TASK [Gathering Facts] ***********************************************************************************************************************************************
ok: [192.168.1.41]

TASK [debug] *********************************************************************************************************************************************************
ok: [192.168.1.41] => {
    "msg": {
        "all_ipv4_addresses": [
            "192.168.1.41"
        ],
        "all_ipv6_addresses": [
            "2400:2411:bc83:3a00:6c56:e3ff:fee5:760c",
            "fe80::6c56:e3ff:fee5:760c"
        ],
        "ansible_local": {},
        "apparmor": {
            "status": "enabled"
        },
        "architecture": "x86_64",
        "bios_date": "04/01/2014",
        "bios_vendor": "SeaBIOS",
        "bios_version": "1.16.3-debian-1.16.3-2",
        "board_asset_tag": "NA",
        "board_name": "NA",
        "board_serial": "NA",
        "board_vendor": "NA",
        "board_version": "NA",
        "chassis_asset_tag": "NA",
        "chassis_serial": "NA",
        "chassis_vendor": "QEMU",
        "chassis_version": "pc-q35-4.2",
        "cmdline": {
            "BOOT_IMAGE": "/vmlinuz-6.8.0-106-generic",
            "console": "ttyS0",
            "ro": true,
            "root": "LABEL=cloudimg-rootfs"
        },
        "date_time": {
            "date": "2026-06-09",
            "day": "09",
            "epoch": "1781036626",
            "epoch_int": "1781036626",
            "hour": "20",
            "iso8601": "2026-06-09T20:23:46Z",
            "iso8601_basic": "20260609T202346644082",
            "iso8601_basic_short": "20260609T202346",
            "iso8601_micro": "2026-06-09T20:23:46.644082Z",
            "minute": "23",
            "month": "06",
            "second": "46",
            "time": "20:23:46",
            "tz": "UTC",
            "tz_dst": "UTC",
            "tz_offset": "+0000",
            "weekday": "Tuesday",
            "weekday_number": "2",
            "weeknumber": "23",
            "year": "2026"
        },
        "default_ipv4": {
            "address": "192.168.1.41",
            "alias": "enp1s0",
            "broadcast": "192.168.1.255",
            "gateway": "192.168.1.1",
            "interface": "enp1s0",
            "macaddress": "6e:56:e3:e5:76:0c",
            "mtu": 1500,
            "netmask": "255.255.255.0",
            "network": "192.168.1.0",
            "prefix": "24",
            "type": "ether"
        },
        "default_ipv6": {
            "address": "2400:2411:bc83:3a00:6c56:e3ff:fee5:760c",
            "gateway": "fe80::e73:29ff:feac:be58",
            "interface": "enp1s0",
            "macaddress": "6e:56:e3:e5:76:0c",
            "mtu": 1500,
            "prefix": "64",
            "scope": "global",
            "type": "ether"
        },
        "device_links": {
            "ids": {
                "sr0": [
                    "ata-QEMU_DVD-ROM_QM00001"
                ]
            },
            "labels": {
                "sr0": [
                    "cidata"
                ],
                "vda1": [
                    "cloudimg-rootfs"
                ],
                "vda15": [
                    "UEFI"
                ],
                "vda16": [
                    "BOOT"
                ]
            },
            "masters": {},
            "uuids": {
                "sr0": [
                    "2026-06-09-20-23-20-00"
                ],
                "vda1": [
                    "6d085073-3c5a-49aa-9133-d0c30d62315b"
                ],
                "vda15": [
                    "A732-B748"
                ],
                "vda16": [
                    "02ae2efa-0a43-47c2-94dd-0cab2e898ba6"
                ]
            }
        },
        "devices": {
            "loop0": {
                "holders": [],
                "host": "",
                "links": {
                    "ids": [],
                    "labels": [],
                    "masters": [],
                    "uuids": []
                },
                "model": null,
                "partitions": {},
                "removable": "0",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "none",
                "sectors": "0",
                "sectorsize": "512",
                "size": "0.00 Bytes",
                "support_discard": "4096",
                "vendor": null,
                "virtual": 1
            },
            "loop1": {
                "holders": [],
                "host": "",
                "links": {
                    "ids": [],
                    "labels": [],
                    "masters": [],
                    "uuids": []
                },
                "model": null,
                "partitions": {},
                "removable": "0",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "none",
                "sectors": "0",
                "sectorsize": "512",
                "size": "0.00 Bytes",
                "support_discard": "512",
                "vendor": null,
                "virtual": 1
            },
            "loop2": {
                "holders": [],
                "host": "",
                "links": {
                    "ids": [],
                    "labels": [],
                    "masters": [],
                    "uuids": []
                },
                "model": null,
                "partitions": {},
                "removable": "0",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "none",
                "sectors": "0",
                "sectorsize": "512",
                "size": "0.00 Bytes",
                "support_discard": "512",
                "vendor": null,
                "virtual": 1
            },
            "loop3": {
                "holders": [],
                "host": "",
                "links": {
                    "ids": [],
                    "labels": [],
                    "masters": [],
                    "uuids": []
                },
                "model": null,
                "partitions": {},
                "removable": "0",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "none",
                "sectors": "0",
                "sectorsize": "512",
                "size": "0.00 Bytes",
                "support_discard": "512",
                "vendor": null,
                "virtual": 1
            },
            "loop4": {
                "holders": [],
                "host": "",
                "links": {
                    "ids": [],
                    "labels": [],
                    "masters": [],
                    "uuids": []
                },
                "model": null,
                "partitions": {},
                "removable": "0",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "none",
                "sectors": "0",
                "sectorsize": "512",
                "size": "0.00 Bytes",
                "support_discard": "512",
                "vendor": null,
                "virtual": 1
            },
            "loop5": {
                "holders": [],
                "host": "",
                "links": {
                    "ids": [],
                    "labels": [],
                    "masters": [],
                    "uuids": []
                },
                "model": null,
                "partitions": {},
                "removable": "0",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "none",
                "sectors": "0",
                "sectorsize": "512",
                "size": "0.00 Bytes",
                "support_discard": "512",
                "vendor": null,
                "virtual": 1
            },
            "loop6": {
                "holders": [],
                "host": "",
                "links": {
                    "ids": [],
                    "labels": [],
                    "masters": [],
                    "uuids": []
                },
                "model": null,
                "partitions": {},
                "removable": "0",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "none",
                "sectors": "0",
                "sectorsize": "512",
                "size": "0.00 Bytes",
                "support_discard": "512",
                "vendor": null,
                "virtual": 1
            },
            "loop7": {
                "holders": [],
                "host": "",
                "links": {
                    "ids": [],
                    "labels": [],
                    "masters": [],
                    "uuids": []
                },
                "model": null,
                "partitions": {},
                "removable": "0",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "none",
                "sectors": "0",
                "sectorsize": "512",
                "size": "0.00 Bytes",
                "support_discard": "512",
                "vendor": null,
                "virtual": 1
            },
            "sr0": {
                "holders": [],
                "host": "SATA controller: Intel Corporation 82801IR/IO/IH (ICH9R/DO/DH) 6 port SATA Controller [AHCI mode] (rev 02)",
                "links": {
                    "ids": [
                        "ata-QEMU_DVD-ROM_QM00001"
                    ],
                    "labels": [
                        "cidata"
                    ],
                    "masters": [],
                    "uuids": [
                        "2026-06-09-20-23-20-00"
                    ]
                },
                "model": "QEMU DVD-ROM",
                "partitions": {},
                "removable": "1",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "mq-deadline",
                "sectors": "736",
                "sectorsize": "2048",
                "size": "368.00 KB",
                "support_discard": "2048",
                "vendor": "QEMU",
                "virtual": 1
            },
            "vda": {
                "holders": [],
                "host": "SCSI storage controller: Red Hat, Inc. Virtio 1.0 block device (rev 01)",
                "links": {
                    "ids": [],
                    "labels": [],
                    "masters": [],
                    "uuids": []
                },
                "model": null,
                "partitions": {
                    "vda1": {
                        "holders": [],
                        "links": {
                            "ids": [],
                            "labels": [
                                "cloudimg-rootfs"
                            ],
                            "masters": [],
                            "uuids": [
                                "6d085073-3c5a-49aa-9133-d0c30d62315b"
                            ]
                        },
                        "sectors": "31455199",
                        "sectorsize": 512,
                        "size": "15.00 GB",
                        "start": "2099200",
                        "uuid": "6d085073-3c5a-49aa-9133-d0c30d62315b"
                    },
                    "vda14": {
                        "holders": [],
                        "links": {
                            "ids": [],
                            "labels": [],
                            "masters": [],
                            "uuids": []
                        },
                        "sectors": "8192",
                        "sectorsize": 512,
                        "size": "4.00 MB",
                        "start": "2048",
                        "uuid": null
                    },
                    "vda15": {
                        "holders": [],
                        "links": {
                            "ids": [],
                            "labels": [
                                "UEFI"
                            ],
                            "masters": [],
                            "uuids": [
                                "A732-B748"
                            ]
                        },
                        "sectors": "217088",
                        "sectorsize": 512,
                        "size": "106.00 MB",
                        "start": "10240",
                        "uuid": "A732-B748"
                    },
                    "vda16": {
                        "holders": [],
                        "links": {
                            "ids": [],
                            "labels": [
                                "BOOT"
                            ],
                            "masters": [],
                            "uuids": [
                                "02ae2efa-0a43-47c2-94dd-0cab2e898ba6"
                            ]
                        },
                        "sectors": "1869825",
                        "sectorsize": 512,
                        "size": "913.00 MB",
                        "start": "227328",
                        "uuid": "02ae2efa-0a43-47c2-94dd-0cab2e898ba6"
                    }
                },
                "removable": "0",
                "rotational": "1",
                "sas_address": null,
                "sas_device_handle": null,
                "scheduler_mode": "mq-deadline",
                "sectors": "33554432",
                "sectorsize": "512",
                "size": "16.00 GB",
                "support_discard": "512",
                "vendor": "0x1af4",
                "virtual": 1
            }
        },
        "discovered_interpreter_python": "/usr/bin/python3",
        "distribution": "Ubuntu",
        "distribution_file_parsed": true,
        "distribution_file_path": "/etc/os-release",
        "distribution_file_variety": "Debian",
        "distribution_major_version": "24",
        "distribution_release": "noble",
        "distribution_version": "24.04",
        "dns": {
            "nameservers": [
                "127.0.0.53"
            ],
            "options": {
                "edns0": true,
                "trust-ad": true
            },
            "search": [
                "labo.local",
                "lab.local"
            ]
        },
        "domain": "",
        "effective_group_id": 0,
        "effective_user_id": 0,
        "enp1s0": {
            "active": true,
            "device": "enp1s0",
            "features": {
                "esp_hw_offload": "off [fixed]",
                "esp_tx_csum_hw_offload": "off [fixed]",
                "fcoe_mtu": "off [fixed]",
                "generic_receive_offload": "on",
                "generic_segmentation_offload": "on",
                "highdma": "on [fixed]",
                "hsr_dup_offload": "off [fixed]",
                "hsr_fwd_offload": "off [fixed]",
                "hsr_tag_ins_offload": "off [fixed]",
                "hsr_tag_rm_offload": "off [fixed]",
                "hw_tc_offload": "off [fixed]",
                "l2_fwd_offload": "off [fixed]",
                "large_receive_offload": "off [fixed]",
                "loopback": "off [fixed]",
                "macsec_hw_offload": "off [fixed]",
                "netns_local": "off [fixed]",
                "ntuple_filters": "off [fixed]",
                "receive_hashing": "off [fixed]",
                "rx_all": "off [fixed]",
                "rx_checksumming": "on [fixed]",
                "rx_fcs": "off [fixed]",
                "rx_gro_hw": "on",
                "rx_gro_list": "off",
                "rx_udp_gro_forwarding": "off",
                "rx_udp_tunnel_port_offload": "off [fixed]",
                "rx_vlan_filter": "on [fixed]",
                "rx_vlan_offload": "off [fixed]",
                "rx_vlan_stag_filter": "off [fixed]",
                "rx_vlan_stag_hw_parse": "off [fixed]",
                "scatter_gather": "on",
                "tcp_segmentation_offload": "on",
                "tls_hw_record": "off [fixed]",
                "tls_hw_rx_offload": "off [fixed]",
                "tls_hw_tx_offload": "off [fixed]",
                "tx_checksum_fcoe_crc": "off [fixed]",
                "tx_checksum_ip_generic": "on",
                "tx_checksum_ipv4": "off [fixed]",
                "tx_checksum_ipv6": "off [fixed]",
                "tx_checksum_sctp": "off [fixed]",
                "tx_checksumming": "on",
                "tx_esp_segmentation": "off [fixed]",
                "tx_fcoe_segmentation": "off [fixed]",
                "tx_gre_csum_segmentation": "off [fixed]",
                "tx_gre_segmentation": "off [fixed]",
                "tx_gso_list": "off [fixed]",
                "tx_gso_partial": "off [fixed]",
                "tx_gso_robust": "on [fixed]",
                "tx_ipxip4_segmentation": "off [fixed]",
                "tx_ipxip6_segmentation": "off [fixed]",
                "tx_lockless": "off [fixed]",
                "tx_nocache_copy": "off",
                "tx_scatter_gather": "on",
                "tx_scatter_gather_fraglist": "off [fixed]",
                "tx_sctp_segmentation": "off [fixed]",
                "tx_tcp6_segmentation": "on",
                "tx_tcp_ecn_segmentation": "on",
                "tx_tcp_mangleid_segmentation": "off",
                "tx_tcp_segmentation": "on",
                "tx_tunnel_remcsum_segmentation": "off [fixed]",
                "tx_udp_segmentation": "off [fixed]",
                "tx_udp_tnl_csum_segmentation": "off [fixed]",
                "tx_udp_tnl_segmentation": "off [fixed]",
                "tx_vlan_offload": "off [fixed]",
                "tx_vlan_stag_hw_insert": "off [fixed]",
                "vlan_challenged": "off [fixed]"
            },
            "hw_timestamp_filters": [],
            "ipv4": {
                "address": "192.168.1.41",
                "broadcast": "192.168.1.255",
                "netmask": "255.255.255.0",
                "network": "192.168.1.0",
                "prefix": "24"
            },
            "ipv6": [
                {
                    "address": "2400:2411:bc83:3a00:6c56:e3ff:fee5:760c",
                    "prefix": "64",
                    "scope": "global"
                },
                {
                    "address": "fe80::6c56:e3ff:fee5:760c",
                    "prefix": "64",
                    "scope": "link"
                }
            ],
            "macaddress": "6e:56:e3:e5:76:0c",
            "mtu": 1500,
            "pciid": "virtio0",
            "promisc": false,
            "speed": -1,
            "timestamping": [],
            "type": "ether"
        },
        "env": {
            "DBUS_SESSION_BUS_ADDRESS": "unix:path=/run/user/0/bus",
            "HOME": "/root",
            "LANG": "C.UTF-8",
            "LC_ADDRESS": "en_US.UTF-8",
            "LC_IDENTIFICATION": "en_US.UTF-8",
            "LC_MEASUREMENT": "en_US.UTF-8",
            "LC_MONETARY": "en_US.UTF-8",
            "LC_NAME": "en_US.UTF-8",
            "LC_NUMERIC": "en_US.UTF-8",
            "LC_PAPER": "en_US.UTF-8",
            "LC_TELEPHONE": "en_US.UTF-8",
            "LC_TIME": "en_US.UTF-8",
            "LOGNAME": "root",
            "PATH": "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin",
            "PWD": "/root",
            "SHELL": "/bin/bash",
            "SHLVL": "0",
            "SSH_CLIENT": "192.168.1.10 43104 22",
            "SSH_CONNECTION": "192.168.1.10 43104 192.168.1.41 22",
            "SSH_TTY": "/dev/pts/0",
            "TERM": "xterm-256color",
            "USER": "root",
            "XDG_RUNTIME_DIR": "/run/user/0",
            "XDG_SESSION_CLASS": "user",
            "XDG_SESSION_ID": "1",
            "XDG_SESSION_TYPE": "tty",
            "_": "/bin/sh"
        },
        "fibre_channel_wwn": [],
        "fips": false,
        "form_factor": "Other",
        "fqdn": "server-41",
        "gather_subset": [
            "all"
        ],
        "hostname": "server-41",
        "hostnqn": "",
        "interfaces": [
            "lo",
            "enp1s0"
        ],
        "is_chroot": false,
        "iscsi_iqn": "",
        "kernel": "6.8.0-106-generic",
        "kernel_version": "#106-Ubuntu SMP PREEMPT_DYNAMIC Fri Mar  6 07:58:08 UTC 2026",
        "lo": {
            "active": true,
            "device": "lo",
            "features": {
                "esp_hw_offload": "off [fixed]",
                "esp_tx_csum_hw_offload": "off [fixed]",
                "fcoe_mtu": "off [fixed]",
                "generic_receive_offload": "on",
                "generic_segmentation_offload": "on",
                "highdma": "on [fixed]",
                "hsr_dup_offload": "off [fixed]",
                "hsr_fwd_offload": "off [fixed]",
                "hsr_tag_ins_offload": "off [fixed]",
                "hsr_tag_rm_offload": "off [fixed]",
                "hw_tc_offload": "off [fixed]",
                "l2_fwd_offload": "off [fixed]",
                "large_receive_offload": "off [fixed]",
                "loopback": "on [fixed]",
                "macsec_hw_offload": "off [fixed]",
                "netns_local": "on [fixed]",
                "ntuple_filters": "off [fixed]",
                "receive_hashing": "off [fixed]",
                "rx_all": "off [fixed]",
                "rx_checksumming": "on [fixed]",
                "rx_fcs": "off [fixed]",
                "rx_gro_hw": "off [fixed]",
                "rx_gro_list": "off",
                "rx_udp_gro_forwarding": "off",
                "rx_udp_tunnel_port_offload": "off [fixed]",
                "rx_vlan_filter": "off [fixed]",
                "rx_vlan_offload": "off [fixed]",
                "rx_vlan_stag_filter": "off [fixed]",
                "rx_vlan_stag_hw_parse": "off [fixed]",
                "scatter_gather": "on",
                "tcp_segmentation_offload": "on",
                "tls_hw_record": "off [fixed]",
                "tls_hw_rx_offload": "off [fixed]",
                "tls_hw_tx_offload": "off [fixed]",
                "tx_checksum_fcoe_crc": "off [fixed]",
                "tx_checksum_ip_generic": "on [fixed]",
                "tx_checksum_ipv4": "off [fixed]",
                "tx_checksum_ipv6": "off [fixed]",
                "tx_checksum_sctp": "on [fixed]",
                "tx_checksumming": "on",
                "tx_esp_segmentation": "off [fixed]",
                "tx_fcoe_segmentation": "off [fixed]",
                "tx_gre_csum_segmentation": "off [fixed]",
                "tx_gre_segmentation": "off [fixed]",
                "tx_gso_list": "on",
                "tx_gso_partial": "off [fixed]",
                "tx_gso_robust": "off [fixed]",
                "tx_ipxip4_segmentation": "off [fixed]",
                "tx_ipxip6_segmentation": "off [fixed]",
                "tx_lockless": "on [fixed]",
                "tx_nocache_copy": "off [fixed]",
                "tx_scatter_gather": "on [fixed]",
                "tx_scatter_gather_fraglist": "on [fixed]",
                "tx_sctp_segmentation": "on",
                "tx_tcp6_segmentation": "on",
                "tx_tcp_ecn_segmentation": "on",
                "tx_tcp_mangleid_segmentation": "on",
                "tx_tcp_segmentation": "on",
                "tx_tunnel_remcsum_segmentation": "off [fixed]",
                "tx_udp_segmentation": "on",
                "tx_udp_tnl_csum_segmentation": "off [fixed]",
                "tx_udp_tnl_segmentation": "off [fixed]",
                "tx_vlan_offload": "off [fixed]",
                "tx_vlan_stag_hw_insert": "off [fixed]",
                "vlan_challenged": "on [fixed]"
            },
            "hw_timestamp_filters": [],
            "ipv4": {
                "address": "127.0.0.1",
                "broadcast": "",
                "netmask": "255.0.0.0",
                "network": "127.0.0.0",
                "prefix": "8"
            },
            "ipv6": [
                {
                    "address": "::1",
                    "prefix": "128",
                    "scope": "host"
                }
            ],
            "mtu": 65536,
            "promisc": false,
            "timestamping": [],
            "type": "loopback"
        },
        "loadavg": {
            "15m": 0.0322265625,
            "1m": 0.431640625,
            "5m": 0.09814453125
        },
        "locally_reachable_ips": {
            "ipv4": [
                "127.0.0.0/8",
                "127.0.0.1",
                "192.168.1.41"
            ],
            "ipv6": [
                "::1",
                "2400:2411:bc83:3a00:6c56:e3ff:fee5:760c",
                "fe80::6c56:e3ff:fee5:760c"
            ]
        },
        "lsb": {
            "codename": "noble",
            "description": "Ubuntu 24.04.4 LTS",
            "id": "Ubuntu",
            "major_release": "24",
            "release": "24.04"
        },
        "lvm": {
            "lvs": {},
            "pvs": {},
            "vgs": {}
        },
        "machine": "x86_64",
        "machine_id": "5da0df7dbf16489695ac634074b1fb3c",
        "memfree_mb": 401,
        "memory_mb": {
            "nocache": {
                "free": 757,
                "used": 204
            },
            "real": {
                "free": 401,
                "total": 961,
                "used": 560
            },
            "swap": {
                "cached": 0,
                "free": 0,
                "total": 0,
                "used": 0
            }
        },
        "memtotal_mb": 961,
        "module_setup": true,
        "mounts": [
            {
                "block_available": 3351260,
                "block_size": 4096,
                "block_total": 3789825,
                "block_used": 438565,
                "device": "/dev/vda1",
                "fstype": "ext4",
                "inode_available": 1880745,
                "inode_total": 1966080,
                "inode_used": 85335,
                "mount": "/",
                "options": "rw,relatime,discard,errors=remount-ro,commit=30",
                "size_available": 13726760960,
                "size_total": 15523123200,
                "uuid": "6d085073-3c5a-49aa-9133-d0c30d62315b"
            },
            {
                "block_available": 193415,
                "block_size": 4096,
                "block_total": 225380,
                "block_used": 31965,
                "device": "/dev/vda16",
                "fstype": "ext4",
                "inode_available": 57896,
                "inode_total": 58496,
                "inode_used": 600,
                "mount": "/boot",
                "options": "rw,relatime",
                "size_available": 792227840,
                "size_total": 923156480,
                "uuid": "02ae2efa-0a43-47c2-94dd-0cab2e898ba6"
            },
            {
                "block_available": 201164,
                "block_size": 512,
                "block_total": 213663,
                "block_used": 12499,
                "device": "/dev/vda15",
                "fstype": "vfat",
                "inode_available": 0,
                "inode_total": 0,
                "inode_used": 0,
                "mount": "/boot/efi",
                "options": "rw,relatime,fmask=0077,dmask=0077,codepage=437,iocharset=iso8859-1,shortname=mixed,errors=remount-ro",
                "size_available": 102995968,
                "size_total": 109395456,
                "uuid": "A732-B748"
            }
        ],
        "nodename": "server-41",
        "os_family": "Debian",
        "pkg_mgr": "apt",
        "proc_cmdline": {
            "BOOT_IMAGE": "/vmlinuz-6.8.0-106-generic",
            "console": [
                "tty1",
                "ttyS0"
            ],
            "ro": true,
            "root": "LABEL=cloudimg-rootfs"
        },
        "processor": [
            "0",
            "GenuineIntel",
            "Intel(R) Core(TM) i9-9900KF CPU @ 3.60GHz"
        ],
        "processor_cores": 1,
        "processor_count": 1,
        "processor_nproc": 1,
        "processor_threads_per_core": 1,
        "processor_vcpus": 1,
        "product_name": "Standard PC (Q35 + ICH9, 2009)",
        "product_serial": "NA",
        "product_uuid": "5da0df7d-bf16-4896-95ac-634074b1fb3c",
        "product_version": "pc-q35-4.2",
        "python": {
            "executable": "/usr/bin/python3",
            "has_sslcontext": true,
            "type": "cpython",
            "version": {
                "major": 3,
                "micro": 3,
                "minor": 12,
                "releaselevel": "final",
                "serial": 0
            },
            "version_info": [
                3,
                12,
                3,
                "final",
                0
            ]
        },
        "python_version": "3.12.3",
        "real_group_id": 0,
        "real_user_id": 0,
        "selinux": {
            "status": "disabled"
        },
        "selinux_python_present": true,
        "service_mgr": "systemd",
        "ssh_host_key_ecdsa_public": "AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTYAAABBBO6lmpc6NHYQAxVR6JSvkUgTtoG8+mcTozlXUwVcF4+k6mpjaGyLfrAgGPB9Ze8A2hMVCrBIttmMcePFreYw1Sw=",
        "ssh_host_key_ecdsa_public_keytype": "ecdsa-sha2-nistp256",
        "ssh_host_key_ed25519_public": "AAAAC3NzaC1lZDI1NTE5AAAAIIrIPFNSwndgiUURYR1KhfKiFbQA3wzaPKQhT+dT7khw",
        "ssh_host_key_ed25519_public_keytype": "ssh-ed25519",
        "ssh_host_key_rsa_public": "AAAAB3NzaC1yc2EAAAADAQABAAABgQDVa5RViQE1MbZnUZtlS+vfiLhncNMmHJU9A+W29zvFlgfVs8HzE4pMK6xctUXFWKqdp0UQJW3S1hHB7U32DtUFdp0EqkVWyNyE2+hMLNiX41ITN7W6eP+kPRjF4A4FudmDdo/PWgLWk5J3xqEdj4h024vPZJtNPn80Yc2/d9FOBG9I06ezlOJkPBbtMkqoU2CB6fi2sZYofO8A42+BkMaceGDSqAbO7L6QthoCZCpeywkBasy3gfpqQb1RcAXEygKNjCbw9mPif7yUQEnLde2fsbztGRYmrkPpXZkLj2AjRhfmr+v4WjIaTjyVp6tPPYvZL3xC/SDoFbgjPAaYNC9HWGAwfXU6kxalDolewtA88+6LDtvzMM5iLJ5Xe6XjW8TCL32rj4lUHMCbyD4NKut4QtOenU4WMrgH8h2S082FMmxS9qfIXzzfqlm4r5KFGXDVB3dl7RYyCDBoTiNAAu0L3RrL3ZIO7CfGI7F16B8Z2QTQ23ePmohLFRRfZ5Eo2dE=",
        "ssh_host_key_rsa_public_keytype": "ssh-rsa",
        "swapfree_mb": 0,
        "swaptotal_mb": 0,
        "system": "Linux",
        "system_capabilities": [],
        "system_capabilities_enforced": "False",
        "system_vendor": "QEMU",
        "uptime_seconds": 20,
        "user_dir": "/root",
        "user_gecos": "root",
        "user_gid": 0,
        "user_id": "root",
        "user_shell": "/bin/bash",
        "user_uid": 0,
        "userspace_architecture": "x86_64",
        "userspace_bits": "64",
        "virtualization_role": "guest",
        "virtualization_tech_guest": [
            "kvm"
        ],
        "virtualization_tech_host": [
            "kvm"
        ],
        "virtualization_type": "kvm"
    }
}

TASK [パッケージのインストール] **************************************************************************************************************************************
changed: [192.168.1.41]

PLAY RECAP ***********************************************************************************************************************************************************
192.168.1.41               : ok=3    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0   

ubuntu@ws1:~/marmot-manifests/09-setup-ansible$ 
```

## ログイン

公開鍵の設定先を root にしたので、root ユーザーを指定してログインします。

```console
ubuntu@ws1:~$ ssh root@192.168.1.41
Warning: Permanently added '192.168.1.41' (ED25519) to the list of known hosts.
Welcome to Ubuntu 24.04.4 LTS (GNU/Linux 6.8.0-106-generic x86_64)

 * Documentation:  https://help.ubuntu.com
＜中略＞

root@server-41:~# dpkg -l |grep gcc
ii  gcc                             4:13.2.0-7ubuntu1                       amd64        GNU C compiler
ii  gcc-13                          13.3.0-6ubuntu2~24.04.1                 amd64        GNU C compiler
ii  gcc-13-base:amd64               13.3.0-6ubuntu2~24.04.1                 amd64        GCC, the GNU Compiler Collection (base package)
ii  gcc-13-x86-64-linux-gnu         13.3.0-6ubuntu2~24.04.1                 amd64        GNU C compiler for the x86_64-linux-gnu architecture
ii  gcc-14-base:amd64               14.2.0-4ubuntu2~24.04.1                 amd64        GCC, the GNU Compiler Collection (base package)
ii  gcc-x86-64-linux-gnu            4:13.2.0-7ubuntu1                       amd64        GNU C compiler for the amd64 architecture
ii  libgcc-13-dev:amd64             13.3.0-6ubuntu2~24.04.1                 amd64        GCC support library (development files)
ii  libgcc-s1:amd64                 14.2.0-4ubuntu2~24.04.1                 amd64        GCC support library
root@server-41:~# 
```

