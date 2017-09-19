package testing

import (
	"errors"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/hyeoncheon/goul"
)

//** adapter implementation for testing

// GeneratorAdapter ...
type GeneratorAdapter struct {
	goul.Adapter
	ID string
}

// Read implements interface Adapter
func (a *GeneratorAdapter) Read(in chan goul.Item, message goul.Message) (chan goul.Item, error) {
	return goul.Launch(a.reader, in, message)
}

// Write implements interface Adapter
func (a *GeneratorAdapter) Write(in chan goul.Item, message goul.Message) (chan goul.Item, error) {
	return goul.Launch(a.writer, in, message)
}

// reader for testing. it generate packet with input item's meta and data
// then pass it as packet itself or raw packet item.
func (a *GeneratorAdapter) reader(in, out chan goul.Item, message goul.Message) {
	defer close(out)
	defer goul.Log(a.GetLogger(), a.ID, "exit")
	goul.Log(a.GetLogger(), a.ID, "reader in looping...")

	for message := range in {
		packet, err := GeneratePacket(string(message.Data()))
		if err != nil {
			return
		}
		switch message.String() {
		case "packet":
			out <- packet
		case "rawpacket":
			out <- &goul.ItemGeneric{Meta: goul.ItemTypeRawPacket, DATA: packet.Data()}
		}
	}
}

// this writer just bypass input data to output channel for testing.
func (a *GeneratorAdapter) writer(in, out chan goul.Item, message goul.Message) {
	defer close(out)
	defer goul.Log(a.GetLogger(), a.ID, "exit")
	goul.Log(a.GetLogger(), a.ID, "writer in looping...")

	for item := range in { // bypass for debugging
		out <- item
	}
	goul.Log(a.GetLogger(), a.ID, "channel closed")
}

// GeneratePacket returns pcap packet generated with given data payload.
func GeneratePacket(data string) (gopacket.Packet, error) {
	var err error

	tcp := layers.TCP{
		SrcPort: layers.TCPPort(1234),
		DstPort: layers.TCPPort(80),
		Seq:     11050,
		Ack:     0,
		RST:     true,
	}
	ip := layers.IPv4{
		Protocol: layers.IPProtocolTCP, // must to decode next
		SrcIP:    net.IP{127, 0, 0, 1},
		DstIP:    net.IP{8, 8, 8, 8},
		Version:  4,
	}
	ether := layers.Ethernet{
		EthernetType: layers.EthernetTypeIPv4, // must to decode next
		SrcMAC:       net.HardwareAddr{0xFF, 0xAA, 0xFA, 0xAA, 0xFF, 0xAA},
		DstMAC:       net.HardwareAddr{0xBD, 0xBD, 0xBD, 0xBD, 0xBD, 0xBD},
	}
	if err := tcp.SetNetworkLayerForChecksum(&ip); err != nil { // must
		return nil, errors.New("SetNetworkLayerForChecksum")
	}

	buf := gopacket.NewSerializeBuffer()
	opt := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	err = gopacket.SerializeLayers(buf, opt,
		&ether,
		&ip,
		&tcp,
		gopacket.Payload([]byte(data)),
	)
	if err != nil {
		return nil, errors.New("SerializeLayers")
	}

	rawPacket := buf.Bytes()
	return gopacket.NewPacket(rawPacket, layers.LayerTypeEthernet, gopacket.Default), nil
}
