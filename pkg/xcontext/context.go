package xcontext

import (
	"context"

	"google.golang.org/grpc/metadata"
)

func GetUserUUID(ctx context.Context) string {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		if useruuid := ctx.Value("user_uuid"); useruuid != nil {
			return useruuid.(string)
		}
		return ""
	}
	v := md.Get("user_uuid")
	if len(v) == 0 {
		return ""
	}
	return v[0]
}
