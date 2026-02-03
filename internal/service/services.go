package service

import (
	"github.com/goku-m/starter/internal/lib/job"
	"github.com/goku-m/starter/internal/repository"
	"github.com/goku-m/starter/internal/server"
)

type Services struct {
	Auth *AuthService
	Job  *job.JobService
	Todo *TodoService
}

func NewServices(s *server.Server, repos *repository.Repositories) (*Services, error) {
	authService := NewAuthService(s)

	// s.Job.SetAuthService(authService)

	// awsClient, err := aws.NewAWS(s)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create AWS client: %w", err)
	// }

	return &Services{
		Job:  s.Job,
		Auth: authService,
		Todo: NewTodoService(s, repos.Todo),
	}, nil
}
