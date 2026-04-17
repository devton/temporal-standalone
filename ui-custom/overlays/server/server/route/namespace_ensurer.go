package route

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.temporal.io/api/operatorservice/v1"
	"go.temporal.io/api/replication/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"

	"github.com/temporalio/ui-server/v2/server/config"
)

const (
	userNamespacePrefix = "usr-"
	defaultRetention    = 30 * 24 * time.Hour
)

func sanitizeNamespaceName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "@", "-at-")
	name = strings.ReplaceAll(name, ".", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ReplaceAll(name, " ", "-")

	var sanitized strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sanitized.WriteRune(r)
		}
	}
	result := sanitized.String()
	result = strings.Trim(result, "-")
	if len(result) > 60 {
		result = result[:60]
	}
	return result
}

type NamespaceEnsurer struct {
	conn *grpc.ClientConn
}

func NewNamespaceEnsurer(conn *grpc.ClientConn) *NamespaceEnsurer {
	return &NamespaceEnsurer{conn: conn}
}

func (ne *NamespaceEnsurer) EnsureNamespace(username string) (string, error) {
	nsName := userNamespacePrefix + sanitizeNamespaceName(username)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := workflowservice.NewWorkflowServiceClient(ne.conn).DescribeNamespace(ctx, &workflowservice.DescribeNamespaceRequest{
		Namespace: nsName,
	})
	if err == nil {
		return nsName, nil
	}

	err = ne.createNamespace(nsName, username)
	if err != nil {
		return "", fmt.Errorf("failed to create namespace %s: %w", nsName, err)
	}

	log.Printf("[NamespaceEnsurer] Created namespace %s for user %s", nsName, username)
	return nsName, nil
}

func (ne *NamespaceEnsurer) createNamespace(nsName, username string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	_, err := operatorservice.NewOperatorServiceClient(ne.conn).CreateNamespace(ctx, &operatorservice.CreateNamespaceRequest{
		Namespace: &replication.NamespaceInfo{
			Name:        nsName,
			State:       replication.NamespaceInfo_REGISTERED,
			Owner:       username,
			Description: fmt.Sprintf("Auto-created namespace for user %s", username),
		},
		Config: &replication.NamespaceConfig{
			WorkflowExecutionRetentionTtl: &defaultRetention,
			HistoryArchivalState:          replication.NamespaceInfo_DISABLED,
			VisibilityArchivalState:       replication.NamespaceInfo_DISABLED,
		},
		ReplicationConfig: &replication.NamespaceReplicationConfig{
			ActiveClusterName: "active",
		},
		IsGlobalNamespace: false,
	})

	return err
}

func HandleEnsureNamespace(ensurer *NamespaceEnsurer) echo.HandlerFunc {
	return func(c echo.Context) error {
		userInfo, ok := c.Get(UserContextKey).(*UserInfo)
		if !ok || userInfo == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
		}

		nsName, err := ensurer.EnsureNamespace(userInfo.Subject)
		if err != nil {
			log.Printf("[EnsureNamespace] Error: %v", err)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"namespace": nsName,
			"user":      userInfo.Subject,
		})
	}
}

func RegisterNamespaceUserRoutes(e *echo.Group, conn *grpc.ClientConn, cfgProvider *config.ConfigProviderWithRefresh) {
	ensurer := NewNamespaceEnsurer(conn)
	e.POST("/user/ensure-namespace", HandleEnsureNamespace(ensurer))
}
