Name:           gophkeeper-client
Version:        %{?version}%{!?version:0.1.0}
Release:        %{?release}%{!?release:1}%{?dist}
Summary:        GophKeeper CLI client
License:        MIT
BuildArch:      x86_64

Source0:        gophkeeper-cli
Source1:        client.yaml

%description
GophKeeper CLI client.

%prep
%build

%install
install -d %{buildroot}/usr/bin
install -m 0755 %{SOURCE0} %{buildroot}/usr/bin/gophkeeper-cli

install -d %{buildroot}/etc/gophkeeper
install -m 0644 %{SOURCE1} %{buildroot}/etc/gophkeeper/client.yaml

%files
/usr/bin/gophkeeper-cli
%config(noreplace) /etc/gophkeeper/client.yaml
