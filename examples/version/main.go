// Package main demonstrates how to use the Proxmox VE API client to retrieve the version information.
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/sergelogvinov/go-proxmox-rest"
)

func main() {
	c, err := proxmox.New(
		proxmox.ClientConfig{},
		proxmox.WithURL(os.Getenv("PROXMOX_URL")),
		// proxmox.WithTokenAuth(os.Getenv("PROXMOX_TOKENID"), os.Getenv("PROXMOX_SECRET")),
		proxmox.WithPasswordAuth(os.Getenv("PROXMOX_USERNAME"), os.Getenv("PROXMOX_PASSWORD")),
		proxmox.WithInsecure(true),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	ctx := context.Background()

	v, err := c.Version(ctx)
	if err != nil {
		log.Printf("getting version: %v", err)
		return
	}
	fmt.Printf("Release: %s, Version: %s, Repoid: %s\n", v.Release, v.Version, v.Repoid)

	status, err := c.Cluster().Status(ctx)
	if err != nil {
		log.Printf("getting cluster status: %v", err)
		return
	}
	fmt.Printf("Cluster status: %+v\n", status)

	pools, err := c.Pools().List(ctx)
	if err != nil {
		log.Printf("getting pools list: %v", err)
		return
	}
	fmt.Printf("Pools: %+v\n", pools)

	pool, err := c.Pools().Get(ctx, "talos-k8s-proxmox")
	if err != nil {
		log.Printf("getting pool: %v", err)
		return
	}
	fmt.Printf("Pool: %+v\n", pool)
}
