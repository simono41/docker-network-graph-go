# Docker Network Graph Go

Dieses Projekt erstellt eine visuelle Darstellung von Docker-Netzwerken und Containern.

## Voraussetzungen

- Linux-System (vorzugsweise ARM64-Architektur)
- Docker
- Internetverbindung

## Installation und Einrichtung

1. Go 1.23.4 herunterladen und installieren:

```bash
wget https://go.dev/dl/go1.23.4.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.23.4.linux-arm64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

2. Überprüfen Sie die Go-Installation:

```bash
go version
```

3. Klonen Sie das Repository:

```bash
git clone https://code.brothertec.eu/simono41/docker-network-graph-go.git
cd docker-network-graph-go
```

4. Bauen Sie das Docker-Image:

```bash
docker build -t code.brothertec.eu/simono41/docker-network-graph-go:latest .
```

## Verwendung

Führen Sie den Container aus, um die Docker-Netzwerkgrafik zu generieren:

```bash
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock code.brothertec.eu/simono41/docker-network-graph-go:latest | dot -Tsvg -o /opt/containers/picture-uploader/uploads/network.svg
```

## Hinweise

- Stellen Sie sicher, dass Sie die neueste Version von Docker installiert haben.
- Für die Ausführung des Containers sind Root-Rechte oder Mitgliedschaft in der Docker-Gruppe erforderlich.
- Die generierte Grafik wird standardmäßig auf der Konsole ausgegeben. Verwenden Sie Umleitungen, um die Ausgabe in eine Datei zu speichern.
- Benutzen sie diese Seite um sich mit den einzelnen Funktionen von graphviz auseinanderzusetzen

Citations:
[1] https://go.dev/dl/go1.23.4.linux-arm64.tar.gz
[2] https://pkg.go.dev/github.com/goccy/go-graphviz@v0.2.9/cgraph#pkg-index
