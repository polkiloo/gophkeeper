Name:           gophkeeper-server
Version:        %{?version}%{!?version:0.1.0}
Release:        %{?release}%{!?release:1}%{?dist}
Summary:        GophKeeper server
License:        MIT
BuildArch:      x86_64

Source0:        gophkeeper-server
Source1:        server.yaml

%description
GophKeeper server.

%prep
%build

%install
install -d %{buildroot}/usr/bin
install -m 0755 %{SOURCE0} %{buildroot}/usr/bin/gophkeeper-server

install -d %{buildroot}/etc/gophkeeper
install -m 0644 %{SOURCE1} %{buildroot}/etc/gophkeeper/server.yaml

%files
/usr/bin/gophkeeper-server
%config(noreplace) /etc/gophkeeper/server.yaml
