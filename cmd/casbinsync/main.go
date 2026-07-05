package main

import (
	"context"
	"fmt"
	"gin-fast/app/global/app"
	"gin-fast/app/service"
	_ "gin-fast/bootstrap"
)

func main() {
	menuIDs := []uint{
		140342, // task scheduler
		140347, // task logs
		140351, // system params
		140355, // area
		140360, // notice
	}

	permissionService := service.NewPermissionService()
	ctx := context.Background()

	for _, menuID := range menuIDs {
		if err := permissionService.UpdateRoleApiPermissionsByMenuID(ctx, menuID); err != nil {
			panic(fmt.Errorf("sync menu %d failed: %w", menuID, err))
		}
		fmt.Printf("synced menu %d\n", menuID)
	}

	checks := []struct {
		subject string
		path    string
		method  string
	}{
		{"user_1", "/api/sysParam/list", "GET"},
		{"user_1", "/api/sysArea/list", "GET"},
		{"user_1", "/api/sysNotice/list", "GET"},
		{"user_1", "/api/sysJobs/list", "GET"},
	}

	for _, check := range checks {
		ok, err := app.CasbinV2.Enforce(check.subject, check.path, check.method)
		if err != nil {
			panic(fmt.Errorf("verify %s %s %s failed: %w", check.subject, check.method, check.path, err))
		}
		fmt.Printf("verify %s %s %s => %v\n", check.subject, check.method, check.path, ok)
	}

	fmt.Println("casbin sync complete")
}
