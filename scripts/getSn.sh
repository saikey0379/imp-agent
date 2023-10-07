#!/bin/bash

if dmidecode -s system-product-name|grep -qEi 'VMware|VirtualBox|KVM|Xen|Parallels|System Product Name';
then
    cat /sys/class/net/$(route -n|grep ^0.0.0.0|awk '{print$NF}')/address
else
    dmidecode -s system-serial-number 2>/dev/null | awk '/^[^#]/ { print $1 }'
fi  
