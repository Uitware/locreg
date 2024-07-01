package cloudflared

import (
	"context"
	"fmt"
	"log"
	"os"

	cloudflare "github.com/cloudflare/cloudflare-go"
)

func StartTunnel() {
	api, err := cloudflare.NewWithAPIToken(os.Getenv("CLOUDFLARE_API_TOKEN"))
	accountId := os.Getenv("CLOUDFLARE_ACCOUNT_ID")
	ctx := context.Background()

	tunnel, err := api.CreateTunnel(ctx, cloudflare.AccountIdentifier(accountId), cloudflare.TunnelCreateParams{
		Name:   "my-tunnel",
		Secret: os.Getenv("CLOUDFLARE_TUNNEL_SECRET"),
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(tunnel.ID)
}
