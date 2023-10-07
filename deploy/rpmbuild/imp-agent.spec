Name:	    imp-agent
Version:    VERSION
Release:    0
Summary:    imp-agent for devices

Group:	    Application/Agent
License:    GPL
BuildRoot:  /root/rpmbuild/BUILDROOT/${name}-${VERSION}
Source:     imp-agent-VERSION.tgz
Prefix:     /usr/local/imp
%description
    Device info collection and os install


%prep
%setup -q

%install
    mkdir -p %{buildroot}/usr/local/
    mkdir -p %{buildroot}/lib/systemd/system/
    cp -rp ./bin/  %{buildroot}/usr/local/imp/
    cp -rp ./conf/ %{buildroot}/usr/local/imp/
    cp deploy/systemd/imp-agent.service %{buildroot}/lib/systemd/system/

%post
    systemctl daemon-reload
%preun
    if [ "$1" = "0" ] ; then
        systemctl stop imp-agent
        systemctl disable imp-agent
    fi

%postun
    if [ "$1" = "0" ] ; then
        rm -f %{prefix}/bin/imp-agent
        rm -f %{prefix}/conf/imp-agent.conf
        rm -rf %{prefix}/scripts/
        rm -f /lib/systemd/system/imp-agent.service
        systemctl daemon-reload
    fi

%clean
    rm -rf %{buildroot}

%files
    %defattr(-,root,root,0755)
    %{prefix}
    /lib/systemd/system/imp-agent.service

%changelog

