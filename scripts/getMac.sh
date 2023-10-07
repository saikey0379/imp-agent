#!/bin/bash
for _dev in /sys/class/net/*/device
do
  _nic=$(cut -d'/' -f5 <<< $_dev);Mac=""
  ip addr |grep " ${_nic}"|grep -q "state UP"&&if [ -d /proc/net/bonding/ ];then
    Mac=$(awk -v n=$(awk '$3=="'"$_nic"'"{print NR}' /proc/net/bonding/*) '{if(NR==n+5){print$NF}}' /proc/net/bonding/* |grep ":")
  else
    Mac=$(cat /sys/class/net/$_nic/address)
  fi

  if [ ! -z $Mac ];then
    echo $Mac;
  fi
done | grep -E "^([0-9A-Fa-f]{2}[:]){5}([0-9A-Fa-f]{2})$"