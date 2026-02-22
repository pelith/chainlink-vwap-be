package service_test

import (
	"errors"
	"flag"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"go.uber.org/goleak"
	"go.uber.org/mock/gomock"

	"vwap/internal/user"
	"vwap/internal/user/mocks"
	"vwap/internal/user/service"
)

var errDB = errors.New("db error")

func TestService_ByID(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("b1c2d3e4-f5a6-7b8c-9d0e-f1a2b3c4d5e6")
	baseTime := time.Date(2024, 4, 23, 20, 56, 36, 0, time.UTC)
	wantUser := user.User{
		ID:        id,
		Address:   "0x1234567890abcdef1234567890abcdef12345678",
		CreatedAt: baseTime,
		UpdatedAt: baseTime,
	}

	tests := []struct {
		name      string
		setupRepo func(ctrl *gomock.Controller) *mocks.MockRepository
		want      user.User
		wantErr   bool
		err       error
	}{
		{
			name: "success",
			setupRepo: func(ctrl *gomock.Controller) *mocks.MockRepository {
				repo := mocks.NewMockRepository(ctrl)
				repo.EXPECT().
					GetUser(gomock.Any(), id).
					Return(wantUser, nil)

				return repo
			},
			want: wantUser,
		},
		{
			name: "error - not found",
			setupRepo: func(ctrl *gomock.Controller) *mocks.MockRepository {
				repo := mocks.NewMockRepository(ctrl)
				repo.EXPECT().
					GetUser(gomock.Any(), id).
					Return(user.User{}, user.ErrNotFound)

				return repo
			},
			wantErr: true,
			err:     user.ErrNotFound,
		},
		{
			name: "error - other",
			setupRepo: func(ctrl *gomock.Controller) *mocks.MockRepository {
				repo := mocks.NewMockRepository(ctrl)
				repo.EXPECT().
					GetUser(gomock.Any(), id).
					Return(user.User{}, errDB)

				return repo
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			svc := service.New(tt.setupRepo(ctrl))

			got, err := svc.ByID(t.Context(), id)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("ByID() failed: %v", err)
				}

				if tt.err != nil && !errors.Is(err, tt.err) {
					t.Errorf("ByID() error = %v, want %v", err, tt.err)
				}

				return
			}

			if tt.wantErr {
				t.Errorf("ByID() expected error")
				return
			}

			if !cmp.Equal(got, tt.want) {
				t.Errorf("ByID() = %v, want %v, diff %v", got, tt.want, cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestMain(m *testing.M) {
	leak := flag.Bool("leak", true, "enable goleak checks")
	flag.Parse()

	code := m.Run()

	if *leak {
		if err := goleak.Find(); err != nil {
			log.Fatalf("goleak detected leaks: %v", err)
		}
	}

	os.Exit(code)
}
