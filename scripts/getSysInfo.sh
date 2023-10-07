#!/usr/bin/env bash
# version 0.1
export PATH=/bin:/sbin:/usr/bin:/usr/sbin
export LC_ALL=C

exec 3>&1
exec &>/dev/null

# serial & manufacturer

SN=$(/bin/bash /usr/local/imp/scripts/getSn.sh)
MANUFACTURER=$(dmidecode -s system-manufacturer|sed "s/Inc.\|Cloud\|New\|Technologies\|Co.\|,\|Ltd.\| //g")
PRODUCT_NAME=$(dmidecode -s system-product-name | awk '/^[^#]/')

# raid
RAID_DEVICE=$(lspci | grep RAID | sed ':a;N;$!ba;s/\n/\\n/g')

# nic
NIC_DEVICE=$(lspci | grep -i net|sed -e "s/(rev.*$//g"|awk '{for(i=1;i<=NF;i++)if(i==1){printf"{\"Id\":\""$1"\",\"Model\":\""}else if(i==NF){printf$i"\"},"}else{printf$i" "}}'|sed 's/,$//')

# oob
OOB_IP=$(ipmitool lan print | awk '/IP Address[[:blank:]]+:/ { print $NF }')

# cpu info
#CPU_MODEL=$(awk -F':' '/model name/ { print $NF; exit }' /proc/cpuinfo)
#CPU_CORE=$(grep -c 'processor' /proc/cpuinfo)
CPU=$(cat /proc/cpuinfo|sed "s/\t//g"|awk -F ":" '$1=="physical id"{print"\"Id\":\""$NF"\"},"};$1=="model name"{printf"\"Model\":\""$NF"\","}'|sort|uniq -c|awk '{for(i=1;i<=NF;i++){if(i==1){printf"{\"Core\":\""$1"\","}else{printf$i}}}'|sed 's/,$//')
CPU_SUM=$(grep -c 'processor' /proc/cpuinfo)
# memory info
MEMORY_SUM=$(dmidecode -t memory | awk '/^[[:blank:]]+Size.*B/ {if($NF=="GB"){sum += $(NF-1)*1024 }else{sum += $(NF-1)}}END{printf("%d", sum/1024)}')
[[ "$MEMORY_SUM" -eq 0 ]] && MEMORY_SUM=$(dmidecode -t memory | awk '/^[[:blank:]]+Size.*GB/ { sum += $(NF-1) } END { printf("%d", sum) }')
[[ "$MEMORY_SUM" -eq 0 ]] && MEMORY_SUM=$(awk '/MemTotal/ { printf("%d"), $2/1024/1000 }' /proc/meminfo)

MEMORY=$(dmidecode -t memory|sed "s/\t//g"|grep -E "^Locator|^Size.*B|Type:"|sed -n '/Size/,+2p'|awk '{if($1=="Size:"){printf"{\"size\":\""$2" "$3"\","} else if($1=="Locator:"){printf"\"Name\":\""$2"\""}else{printf",\"Type\":\""$2"\"},"}}'| sed 's/,$//')
[[ -z "$MEMORY" ]] && MEMORY=$(dmidecode -t memory | awk -F'[: ]' '/^[[:blank:]]+Size.*GB/ { printf("{\"Name\":\"\",\"size\":\"%s GB\"},", $(NF-1)) }' | sed 's/,$//')

# disk info
if lspci | grep -q 'RAID bus controller.*MegaRAID'; then
    rpm -q MegaCli || rpm -ivh http://imp.example.com/www/packages/MegaCLI/Linux/MegaCli-8.07.14-1.noarch.rpm
    DISK=$(/opt/MegaRAID/MegaCli/MegaCli64 -PDList -aALL -NoLog|awk '$1=="Inquiry"{print$3};$1=="Slot"{printf"{\"slot\":"$NF};$1=="Raw"{printf",\"size\":\""$3$4};$1=="Firmware"{printf"\",\"status\":\""$3"\",\"block\":\""}'|while read i ;do lsblk --nodeps -no name,serial|while read j;do sn=`echo $j|awk '{print$2}'`;block=`echo $j|awk '{print$1}'`;echo $i | grep -q $sn&&echo $i | sed "s/${sn}.*$/$block\"},/g"|awk '{printf$1}'&&break;done||echo -n $i | sed "s/,\"block.*$/},/g";done|sed -e "s/,$//g")
#    DISK=$(/opt/MegaRAID/MegaCli/MegaCli64 -PDList -aALL -NoLog| awk '$1=="Slot"{printf"{\"Name\":\"Slot"$NF"\","};$1=="Raw"{printf("\"size\":\"%s %s\"},", $3, $4) }' | sed 's/,$//')
#    DISK_SUM=$(/opt/MegaRAID/MegaCli/MegaCli64 -PDList -aALL -NoLog| awk '/Raw Size/ { if ($4 == "TB") { sum0 += $3 * 1000 } else { sum1 += $3 } } END { printf("%d", sum0 + sum1) }')
elif lspci | grep -q 'RAID bus controller.*Hewlett-Packard'; then
    rpm -q hpssacli || rpm -ivh http://imp.example.com/www/packages/hpssacli-2.40-13.0.x86_64.rpm
    _slot=$(/usr/sbin/hpssacli ctrl all show status | awk '/Slot/ { print $6 }' | sed q)
    DISK=$(/usr/sbin/hpssacli ctrl slot=$_slot pd all show status | awk '{ sub(/\):/, "", $8); printf("{\"Name\":\"\",\"size\":\"%s %s\"},", $7, $8) }' | sed 's/,$//')
#    DISK_SUM=$(/usr/sbin/hpssacli ctrl slot=$_slot pd all show status | awk '{ if ($8 ~ /TB/) { sum0 += $7 * 1000 } else { sum1 += $7 } } END { printf("%d", sum0 + sum1) }')
else
    DISK=$(fdisk -lu | awk '/^Disk.*bytes/ { gsub(/,/, ""); printf("{\"Name\":\"%s\",\"size\":\"%s %s\"},", $2, $3, $4) }' | sed 's/,$//')
#    DISK_SUM=$(fdisk -lu | awk '/^Disk.*bytes/ { sum += $3 } END { printf("%d", sum) }')
fi
DISK_SUM=$(fdisk -lu | awk '/^Disk.*bytes/ { sum += $3 } END { printf("%d", sum) }')

# network info
NIC=""
if [ -d /proc/net/bonding/ ];then
    for _bond in /proc/net/bonding/*
    do
    	_nic=$(cut -d'/' -f5 <<< $_bond)
    	_mac=$(cat /sys/class/net/$_nic/address)
        _ip=$(ifconfig $_nic | awk '/inet / { gsub(/\/.*/, ""); print $2 }')
        NIC=${NIC}$(echo -n "{\"Name\":\"$_nic\",\"Mac\":\"$_mac\",\"Ip\":\"$_ip\"},")
        NIC=${NIC}$(awk '$1=="Slave"&&NF==3{printf"{\"Name\":\"'%s'\",",$NF};$1=="Permanent"{printf"\"Mac\":\""$NF"\",\"Ip\":\"'$_ip'\"},"}' ${_bond})
    done
fi

for _dev in /sys/class/net/*/device
do
    _nic=$(cut -d'/' -f5 <<< $_dev)
    if ! echo $NIC|grep -q $_nic;then
        _mac=$(cat /sys/class/net/$_nic/address)
        _ip=$(ifconfig $_nic | awk '/inet / { gsub(/\/.*/, ""); print $2 }')
        NIC=${NIC}$(echo -n "{\"Name\":\"$_nic\",\"Mac\":\"$_mac\",\"Ip\":\"$_ip\"},")
    fi
done
NIC=$(echo $NIC|sed 's/,$//')

# is vm
dmidecode | grep -qEi 'VMware|VirtualBox|KVM|Xen|Parallels' && IS_VM=Yes || IS_VM=No
GPU_DEVICE=$(nvidia-smi -L |sed -e "s/(.*$//g"|awk '{cmd="nvidia-smi -i "NR-1"|awk '\''NF>1{if($(NF-1)~\"Default\"){printf$(NF-4)}}'\''";for(i=1;i<=NF;i++){if(i==1){printf"{\"Id\":\""NR-1"\",\"model\":\""}else if(i>1&&i<NF){printf$(i+1)" "}else{printf"\",\"Memory\":\"";system(cmd);printf"\"},"}}}'|sed -e 's/\ ","Memory/","Memory/g' -e 's/,$//')

VER_AGENT=$(rpm -qa | grep imp-agent|awk -F "-" '{print$3}')

# return json
cat >&3 <<EOF
{"Sn":"$SN","Company":"$MANUFACTURER","ModelName":"$PRODUCT_NAME","Motherboard":{"Name":"","Model":""},"Raid":"$RAID_DEVICE","NicDevice":[$NIC_DEVICE],"Oob":"$OOB_IP","Cpu":[$CPU],"CpuSum":$CPU_SUM,"Memory":[$MEMORY],"MemorySum":$MEMORY_SUM,"DiskSum":$DISK_SUM,"Nic":[$NIC],"Disk":[$DISK],"GPU":[$GPU_DEVICE],"IsVm":"$IS_VM","VersionAgt":"$VER_AGENT"}
EOF
