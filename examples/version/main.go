// Package main demonstrates how to use the Proxmox VE API client to retrieve the version information.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sergelogvinov/proxmox/go-proxmox-rest"
)

func main() {
	c, err := proxmox.New(
		proxmox.ClientConfig{},
		proxmox.WithBaseURL(os.Getenv("PROXMOX_URL")),
		proxmox.WithTokenAuth(os.Getenv("PROXMOX_TOKENID"), os.Getenv("PROXMOX_SECRET")),
		// proxmox.WithPasswordAuth(os.Getenv("PROXMOX_USERNAME"), os.Getenv("PROXMOX_PASSWORD")),
		proxmox.WithInsecure(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	v, err := c.Version(context.Background())
	if err != nil {
		log.Printf("getting version: %v", err)
		return
	}

	fmt.Printf("Release: %s, Version: %s, Repoid: %s\n", v.Release, v.Version, v.Repoid)
}
