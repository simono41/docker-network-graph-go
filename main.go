package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"golang.org/x/net/context"
)

// COLORS enthält eine Liste von Farben für die Netzwerke
var COLORS = []string{
	"#1f78b4", "#33a02c", "#e31a1c", "#ff7f00", "#6a3d9a", "#b15928",
	"#a6cee3", "#b2df8a", "#fdbf6f", "#cab2d6", "#90f530", "#0d8bad",
	"#e98420", "#0e9997", "#6a5164", "#afa277", "#149ead", "#a54a56",
}

var i = 0

// Network repräsentiert ein Docker-Netzwerk
type Network struct {
	Name     string
	Gateway  string
	Internal bool
	Isolated bool
	Color    string
}

// Interface repräsentiert eine Netzwerkschnittstelle eines Containers
type Interface struct {
	EndpointID string
	Address    string
	Aliases    []string
}

// Port repräsentiert einen Port eines Containers
type Port struct {
	Port string
}

// Container repräsentiert einen Docker-Container
type Container struct {
	ContainerID string
	Name        string
	Interfaces  []Interface
	Ports       []Port
}

// Link repräsentiert eine Verbindung zwischen einem Container und einem Netzwerk
type Link struct {
	ContainerID string
	EndpointID  string
	NetworkName string
}

func getUniqueColor() string {
	if i < len(COLORS) {
		c := COLORS[i]
		i++
		return c
	}
	return fmt.Sprintf("#%06x", rand.Intn(0xFFFFFF))
}

func getNetworks(cli *client.Client, verbose bool) (map[string]Network, error) {
	networks := make(map[string]Network)
	ctx := context.Background()

	netList, err := cli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return nil, err
	}

	for _, net := range netList {
		gateway := ""
		if len(net.IPAM.Config) > 0 {
			gateway = net.IPAM.Config[0].Subnet
		}

		if gateway == "" {
			continue
		}

		internal := net.Internal
		isolated := false
		if val, ok := net.Options["com.docker.network.bridge.enable_icc"]; ok && val == "false" {
			isolated = true
		}

		if verbose {
			fmt.Printf("Network: %s %s %s gw:%s\n", net.Name,
				map[bool]string{true: "internal", false: ""}[internal],
				map[bool]string{true: "isolated", false: ""}[isolated],
				gateway)
		}

		color := getUniqueColor()
		networks[net.Name] = Network{net.Name, gateway, internal, isolated, color}
	}

	networks["host"] = Network{"host", "0.0.0.0", false, false, "#808080"}
	return networks, nil
}

func getContainers(cli *client.Client, verbose bool) ([]Container, []Link, error) {
	var containers []Container
	var links []Link
	ctx := context.Background()

	contList, err := cli.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return nil, nil, err
	}

	for _, cont := range contList {
		var interfaces []Interface
		var ports []Port

		contInspect, err := cli.ContainerInspect(ctx, cont.ID)
		if err != nil {
			return nil, nil, err
		}

		for portName := range contInspect.NetworkSettings.Ports {
			ports = append(ports, Port{portName.Port()})
		}

		for netName, netInfo := range contInspect.NetworkSettings.Networks {
			aliases := []string{}
			for _, alias := range netInfo.Aliases {
				if alias != cont.ID[:12] && alias != cont.Names[0][1:] {
					aliases = append(aliases, alias)
				}
			}

			interfaces = append(interfaces, Interface{
				EndpointID: netInfo.EndpointID,
				Address:    netInfo.IPAddress,
				Aliases:    aliases,
			})

			links = append(links, Link{
				ContainerID: cont.ID,
				EndpointID:  netInfo.EndpointID,
				NetworkName: netName,
			})
		}

		if verbose {
			fmt.Printf("Container: %s %v %s\n", cont.Names[0][1:], ports, interfaces)
		}

		containers = append(containers, Container{
			ContainerID: cont.ID,
			Name:        cont.Names[0][1:],
			Interfaces:  interfaces,
			Ports:       ports,
		})
	}

	return containers, links, nil
}

func drawNetwork(graph *cgraph.Graph, net Network) error {
	label := fmt.Sprintf("{%s", net.Name)
	if net.Internal {
		label += " | Internal"
	}
	if net.Isolated {
		label += " | Containers isolated"
	}
	label += "}"

	n, err := graph.CreateNodeByName(net.Name)
	if err != nil {
		return err
	}
	n.SetShape(cgraph.BoxShape)
	n.SetLabel(label)
	n.SetColor(net.Color + "60")
	n.SetStyle("rounded")
	return nil
}

func drawContainer(graph *cgraph.Graph, c Container) error {
	var ifaceLabels []string
	var portLabels []string

	for _, port := range c.Ports {
		portLabels = append(portLabels, fmt.Sprintf("{%s}}", port.Port))
	}

	for _, iface := range c.Interfaces {
		ifaceLabel := "{"
		for _, alias := range iface.Aliases {
			ifaceLabel += fmt.Sprintf(" %s |", alias)
		}
		ifaceLabel += fmt.Sprintf("<%s> %s }}", iface.EndpointID, iface.Address)
		ifaceLabels = append(ifaceLabels, ifaceLabel)
	}

	label := fmt.Sprintf("{{ %s ", c.Name)
	if len(portLabels) > 0 {
		label += fmt.Sprintf("| {{ %s }} ", strings.Join(portLabels, " | "))
	}
	label += fmt.Sprintf("| {{ %s }} }}", strings.Join(ifaceLabels, " | "))

	n, err := graph.CreateNodeByName(c.ContainerID)
	if err != nil {
		return err
	}
	n.SetShape(cgraph.BoxShape)
	n.SetLabel(label)
	n.SetFillColor("#cdcdcd")
	n.SetStyle("filled")
	return nil
}

func drawLink(graph *cgraph.Graph, networks map[string]Network, link Link) error {
	style := cgraph.SolidEdgeStyle
	if networks[link.NetworkName].Isolated {
		style = cgraph.DashedEdgeStyle
	} else if networks[link.NetworkName].Name == "host" {
		style = cgraph.BoldEdgeStyle
	}

        n, err := graph.NodeByName(link.ContainerID)
        if err != nil {
            return err
        }

        m, err := graph.NodeByName(link.NetworkName)
        if err != nil {
            return err
        }

        e, err := graph.CreateEdgeByName("", n, m)
        if err != nil {
            return err
        }

	e.SetColor(networks[link.NetworkName].Color)
	e.SetStyle(style)
	return nil
}

func generateGraph(verbose bool, file string, url bool) error {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	networks, err := getNetworks(cli, verbose)
	if err != nil {
		return err
	}

	containers, links, err := getContainers(cli, verbose)
	if err != nil {
		return err
	}

        ctx := context.Background()
	g, err := graphviz.New(ctx)
	if err != nil {
		return err
	}
	graph, err := g.Graph()
	if err != nil {
		return err
	}
	graph.SetLayout("sfdp")
	graph.SetRankDir("LR")
	//graph.SetBackground("transparent")

	for _, network := range networks {
		if err := drawNetwork(graph, network); err != nil {
			return err
		}
	}

	for _, container := range containers {
		if err := drawContainer(graph, container); err != nil {
			return err
		}
	}

	for _, link := range links {
		if link.NetworkName != "none" {
			if err := drawLink(graph, networks, link); err != nil {
				return err
			}
		}
	}

	for _, network := range networks {
		if !network.Internal && network.Name != "host" {
                        startNode, err := graph.CreateNodeByName("start")
			if err != nil {
				return err
			}
                        endNode, err := graph.CreateNodeByName("end")
			if err != nil {
				return err
			}
                        e, err := graph.CreateEdgeByName("e", startNode, endNode)
			if err != nil {
				return err
			}
			e.SetColor("#808080")
			e.SetStyle("dotted")
		}
	}

	if file != "" {
		if err := g.RenderFilename(ctx, graph, graphviz.Format(filepath.Ext(file)[1:]), file); err != nil {
			return err
		}
	} else if url {
		// URL-Generierung ist in dieser Go-Version nicht implementiert
		fmt.Println("URL generation is not implemented in this Go version")
	} else {
		if err := g.Render(ctx, graph, graphviz.Format("dot"), os.Stdout); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	verbose := false
	file := ""
	url := false

	// Einfache Argumentverarbeitung (kann durch ein Argument-Parsing-Paket ersetzt werden)
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-v", "--verbose":
			verbose = true
		case "-o", "--out":
			if i+1 < len(os.Args) {
				file = os.Args[i+1]
				i++
			}
		case "-u", "--url":
			url = true
		}
	}

	if err := generateGraph(verbose, file, url); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
