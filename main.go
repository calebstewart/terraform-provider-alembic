package main

import (
	"context"
	"log"

	"github.com/calebstewart/terraform-provider-alembic/alembic"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

func main() {
	opts := providerserver.ServeOpts{
		// TODO: Update this string with the published name of your provider
		Address: "github.com/calebstewart/alembic",
	}

	err := providerserver.Serve(context.Background(), alembic.New, opts)
	if err != nil {
		log.Fatal(err.Error())
	}
}
