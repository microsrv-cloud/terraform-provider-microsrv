// Package main provides a cleanup janitor for orphaned acceptance test resources.
// It searches for and deletes any project or SSH key starting with "tf-acc-".
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/microsrv-cloud/terraform-provider-microsrv/internal/client"
)

func main() {
	endpoint := os.Getenv("MICROSRV_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.microsrv.ru"
	}

	token := os.Getenv("MICROSRV_TOKEN")
	if token == "" {
		token = os.Getenv("MICROSRV_API_KEY")
	}
	if token == "" {
		log.Println("Neither MICROSRV_TOKEN nor MICROSRV_API_KEY is set; skipping sweep")
		return
	}

	c, err := client.New(client.Config{
		Endpoint:       endpoint,
		Token:          token,
		RequestTimeout: 30 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to initialize client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("=== Sweeping stale acceptance test resources (prefix: tf-acc-) ===")

	// 1. Clean up stale SSH keys
	keys, err := c.ListSSHKeys(ctx)
	if err != nil {
		log.Printf("Warning: failed to list SSH keys: %v\n", err)
	} else {
		for _, k := range keys {
			if strings.HasPrefix(k.Name, "tf-acc-") {
				fmt.Printf("Deleting stale SSH key: %s (%s)... ", k.Name, k.ID)
				if err := c.DeleteSSHKey(ctx, k.ID); err != nil {
					fmt.Printf("failed: %v\n", err)
				} else {
					fmt.Println("done")
				}
			}
		}
	}

	// 2. Clean up stale projects (and their child containers, images, and routes)
	projects, err := c.ListProjects(ctx)
	if err != nil {
		log.Printf("Warning: failed to list projects: %v\n", err)
	} else {
		for _, p := range projects {
			if strings.HasPrefix(p.Name, "tf-acc-") {
				fmt.Printf("Cleaning up stale project: %s (%s)...\n", p.Name, p.ID)

				// Clean up gateway routes in project
				if routes, err := c.ListGatewayRoutes(ctx, p.ID, "", ""); err == nil {
					for _, r := range routes {
						_ = c.DeleteGatewayRoute(ctx, r.ID)
					}
				}

				// Clean up containers in project
				if containers, err := c.ListContainers(ctx, p.ID, ""); err == nil {
					for _, ct := range containers {
						_ = c.DeleteContainer(ctx, ct.ID)
					}
				}

				// Clean up container images in project
				if images, err := c.ListContainerImages(ctx, p.ID, ""); err == nil {
					for _, img := range images {
						_ = c.DeleteContainerImage(ctx, img.ID)
					}
				}

				// Delete the project
				fmt.Printf("Deleting project %s... ", p.ID)
				if err := c.DeleteProject(ctx, p.ID); err != nil {
					fmt.Printf("failed: %v\n", err)
				} else {
					fmt.Println("done")
				}
			}
		}
	}

	fmt.Println("=== Sweep completed ===")
}
