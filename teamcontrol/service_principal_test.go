package teamcontrol

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlannerServicePrincipalCannotLogin(t *testing.T) {
	tests := []struct {
		name string
		run  func(*Service) error
	}{
		{
			name: "regular user creation",
			run: func(service *Service) error {
				_, err := service.CreateUser(CreateUserInput{
					ID:          PlannerServicePrincipal,
					DisplayName: "Planner Service",
				})
				return err
			},
		},
		{
			name: "bootstrap user creation",
			run: func(service *Service) error {
				_, _, err := service.BootstrapFirstUser(
					CreateUserInput{
						ID:          PlannerServicePrincipal,
						DisplayName: "Planner Service",
					},
					"forbidden",
					"planner-service-token-123456",
				)
				return err
			},
		},
		{
			name: "access token issuance",
			run: func(service *Service) error {
				owner, err := service.CreateUser(CreateUserInput{
					ID:          "owner",
					DisplayName: "Owner",
				})
				if err != nil {
					return err
				}
				_, err = service.RegisterAccessToken(
					owner.ID,
					PlannerServicePrincipal,
					"forbidden",
					"planner-service-token-123456",
					nil,
				)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			service, err := Open(t.TempDir())
			require.NoError(t, err)
			require.ErrorContains(
				t,
				test.run(service),
				"non-login service principal",
			)
		})
	}
}
