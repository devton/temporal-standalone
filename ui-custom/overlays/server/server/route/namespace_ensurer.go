package route

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/temporalio/ui-server/v2/server/config"
)

const (
	defaultRetention = 30 * 24 * time.Hour
)

type NamespaceEnsurer struct {
	conn *grpc.ClientConn
}

func NewNamespaceEnsurer(conn *grpc.ClientConn) *NamespaceEnsurer {
	return &NamespaceEnsurer{conn: conn}
}

func (ne *NamespaceEnsurer) findUserNamespace(ownerEmail string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := workflowservice.NewWorkflowServiceClient(ne.conn)
	resp, err := client.ListNamespaces(ctx, &workflowservice.ListNamespacesRequest{
		PageSize: 100,
	})
	if err != nil {
		return "", fmt.Errorf("list namespaces failed: %w", err)
	}

	for _, ns := range resp.Namespaces {
		if ns.NamespaceInfo == nil {
			continue
		}
		if ns.NamespaceInfo.OwnerEmail == ownerEmail {
			if ns.NamespaceInfo.Data != nil {
				if ns.NamespaceInfo.Data["type"] == "primary" {
					return ns.NamespaceInfo.Name, nil
				}
			}
		}
	}

	return "", nil
}

func (ne *NamespaceEnsurer) createNamespace(name, ownerEmail, nsType, description string) (string, error) {
	if name == "" {
		name = uuid.New().String()
	}

	if description == "" {
		description = fmt.Sprintf("Auto-created %s namespace for %s", nsType, ownerEmail)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	data := map[string]string{
		"type":  nsType,
		"owner": ownerEmail,
	}

	_, err := workflowservice.NewWorkflowServiceClient(ne.conn).RegisterNamespace(ctx, &workflowservice.RegisterNamespaceRequest{
		Namespace:                        name,
		Description:                      description,
		OwnerEmail:                       ownerEmail,
		WorkflowExecutionRetentionPeriod: durationpb.New(defaultRetention),
		IsGlobalNamespace:                false,
		Data:                             data,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create namespace %s: %w", name, err)
	}

	log.Printf("[NamespaceEnsurer] Created namespace %s (%s) for %s", name, nsType, ownerEmail)
	return name, nil
}

func (ne *NamespaceEnsurer) EnsureNamespace(ownerEmail string) (string, error) {
	existing, err := ne.findUserNamespace(ownerEmail)
	if err != nil {
		log.Printf("[NamespaceEnsurer] findUserNamespace error: %v", err)
	}
	if existing != "" {
		return existing, nil
	}

	nsName := uuid.New().String()
	created, err := ne.createNamespace(nsName, ownerEmail, "primary", "")
	if err != nil {
		return "", err
	}
	return created, nil
}

func (ne *NamespaceEnsurer) ListUserNamespaces(ownerEmail string) ([]map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := workflowservice.NewWorkflowServiceClient(ne.conn)
	resp, err := client.ListNamespaces(ctx, &workflowservice.ListNamespacesRequest{
		PageSize: 100,
	})
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for _, ns := range resp.Namespaces {
		if ns.NamespaceInfo == nil {
			continue
		}
		if ns.NamespaceInfo.OwnerEmail == ownerEmail {
			nsType := "custom"
			if ns.NamespaceInfo.Data != nil {
				if t, ok := ns.NamespaceInfo.Data["type"]; ok {
					nsType = t
				}
			}
			result = append(result, map[string]interface{}{
				"name":        ns.NamespaceInfo.Name,
				"type":        nsType,
				"description": ns.NamespaceInfo.Description,
				"state":       ns.NamespaceInfo.State.String(),
			})
		}
	}

	return result, nil
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

func HandleListMyNamespaces(ensurer *NamespaceEnsurer) echo.HandlerFunc {
	return func(c echo.Context) error {
		userInfo, ok := c.Get(UserContextKey).(*UserInfo)
		if !ok || userInfo == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
		}

		namespaces, err := ensurer.ListUserNamespaces(userInfo.Subject)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusOK, map[string]interface{}{
			"namespaces": namespaces,
		})
	}
}

type CreateNamespaceRequest struct {
	Description string `json:"description"`
}

func HandleCreateNamespace(ensurer *NamespaceEnsurer) echo.HandlerFunc {
	return func(c echo.Context) error {
		userInfo, ok := c.Get(UserContextKey).(*UserInfo)
		if !ok || userInfo == nil {
			return echo.NewHTTPError(http.StatusUnauthorized, "not authenticated")
		}

		var req CreateNamespaceRequest
		if err := c.Bind(&req); err != nil {
			req.Description = ""
		}

		nsName := uuid.New().String()
		desc := req.Description
		if desc == "" {
			desc = fmt.Sprintf("Custom namespace for %s", userInfo.Subject)
		}

		created, err := ensurer.createNamespace(nsName, userInfo.Subject, "custom", desc)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}

		return c.JSON(http.StatusCreated, map[string]interface{}{
			"namespace":   created,
			"type":        "custom",
			"description": desc,
		})
	}
}

func RegisterNamespaceUserRoutes(e *echo.Group, conn *grpc.ClientConn, cfgProvider *config.ConfigProviderWithRefresh) {
	ensurer := NewNamespaceEnsurer(conn)
	authMW := AuthMiddleware(cfgProvider)
	user := e.Group("/user", authMW)
	user.POST("/ensure-namespace", HandleEnsureNamespace(ensurer))
	user.GET("/namespaces", HandleListMyNamespaces(ensurer))
	user.POST("/namespaces", HandleCreateNamespace(ensurer))
}
