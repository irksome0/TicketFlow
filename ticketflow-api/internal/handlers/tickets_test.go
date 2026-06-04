package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"ticketflow-api/internal/models"
	"ticketflow-api/internal/repository"
)

type fakeTicketRepo struct {
	tickets      map[uuid.UUID]*models.Ticket
	comments     map[uuid.UUID][]models.TicketComment
	findErr      error
	updateCalled bool
	assignCalled bool
	assignedID   *uuid.UUID
}

func (r *fakeTicketRepo) Create(ticket *models.Ticket) error {
	r.tickets[ticket.ID] = ticket
	return nil
}

func (r *fakeTicketRepo) FindByID(id uuid.UUID) (*models.Ticket, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	ticket, ok := r.tickets[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return ticket, nil
}

func (r *fakeTicketRepo) ListByOrganization(uuid.UUID, repository.TicketFilter) ([]models.Ticket, error) {
	return nil, nil
}

func (r *fakeTicketRepo) UpdateStatus(uuid.UUID, models.TicketStatus, models.TicketStatus, *time.Time, uuid.UUID) error {
	r.updateCalled = true
	return nil
}

func (r *fakeTicketRepo) ListStatusHistory(uuid.UUID) ([]models.TicketStatusHistory, error) {
	return nil, nil
}

func (r *fakeTicketRepo) AssignTo(id uuid.UUID, assigneeID *uuid.UUID) error {
	r.assignCalled = true
	r.assignedID = assigneeID
	if ticket, ok := r.tickets[id]; ok {
		ticket.AssigneeID = assigneeID
	}
	return nil
}

func (r *fakeTicketRepo) CreateComment(comment *models.TicketComment) error {
	if comment.CreatedAt.IsZero() {
		comment.CreatedAt = time.Now().UTC()
	}
	r.comments[comment.TicketID] = append(r.comments[comment.TicketID], *comment)
	return nil
}

func (r *fakeTicketRepo) ListComments(ticketID uuid.UUID) ([]models.TicketComment, error) {
	return r.comments[ticketID], nil
}

type fakeUserRepo struct {
	users map[uuid.UUID]*models.User
}

func (r *fakeUserRepo) Create(*models.User) error {
	return nil
}

func (r *fakeUserRepo) FindByEmail(string) (*models.User, error) {
	return nil, gorm.ErrRecordNotFound
}

func (r *fakeUserRepo) FindByID(id uuid.UUID) (*models.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	return user, nil
}

func (r *fakeUserRepo) ListByOrganization(orgID uuid.UUID) ([]models.User, error) {
	users := make([]models.User, 0, len(r.users))
	for _, user := range r.users {
		if user.OrganizationID == orgID {
			users = append(users, *user)
		}
	}
	return users, nil
}

func (r *fakeUserRepo) UpdateRole(uuid.UUID, models.Role) error {
	return nil
}

type fakeAttachmentRepo struct {
	attachment *models.Attachment
}

func (r *fakeAttachmentRepo) Create(*models.Attachment) error {
	return nil
}

func (r *fakeAttachmentRepo) ListByTicketID(uuid.UUID) ([]models.Attachment, error) {
	return nil, nil
}

func (r *fakeAttachmentRepo) FindByID(uuid.UUID) (*models.Attachment, error) {
	if r.attachment == nil {
		return nil, gorm.ErrRecordNotFound
	}
	return r.attachment, nil
}

type fakeStorage struct {
	opened bool
}

func (s *fakeStorage) Save(context.Context, string, io.Reader, string) error {
	return nil
}

func (s *fakeStorage) Open(context.Context, string) (io.ReadCloser, error) {
	s.opened = true
	return io.NopCloser(strings.NewReader("file")), nil
}

func (s *fakeStorage) Delete(context.Context, string) error {
	return nil
}

func TestCreateCommentAccessRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgID := uuid.New()
	otherOrgID := uuid.New()
	clientID := uuid.New()
	otherClientID := uuid.New()
	operatorID := uuid.New()
	ticketID := uuid.New()

	tests := []struct {
		name       string
		userID     uuid.UUID
		orgID      uuid.UUID
		role       models.Role
		ticket     *models.Ticket
		body       string
		wantStatus int
	}{
		{
			name:       "client can comment own ticket",
			userID:     clientID,
			orgID:      orgID,
			role:       models.RoleClient,
			ticket:     &models.Ticket{ID: ticketID, OrganizationID: orgID, CreatorID: clientID},
			body:       `{"message":"Need additional help"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "client cannot comment another client ticket",
			userID:     clientID,
			orgID:      orgID,
			role:       models.RoleClient,
			ticket:     &models.Ticket{ID: ticketID, OrganizationID: orgID, CreatorID: otherClientID},
			body:       `{"message":"Need additional help"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "operator can comment organization ticket",
			userID:     operatorID,
			orgID:      orgID,
			role:       models.RoleOperator,
			ticket:     &models.Ticket{ID: ticketID, OrganizationID: orgID, CreatorID: clientID},
			body:       `{"message":"Work started"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name:       "other organization cannot comment ticket",
			userID:     operatorID,
			orgID:      otherOrgID,
			role:       models.RoleOperator,
			ticket:     &models.Ticket{ID: ticketID, OrganizationID: orgID, CreatorID: clientID},
			body:       `{"message":"Work started"}`,
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "empty comment is rejected",
			userID:     clientID,
			orgID:      orgID,
			role:       models.RoleClient,
			ticket:     &models.Ticket{ID: ticketID, OrganizationID: orgID, CreatorID: clientID},
			body:       `{"message":"   "}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &fakeTicketRepo{
				tickets:  map[uuid.UUID]*models.Ticket{ticketID: tt.ticket},
				comments: map[uuid.UUID][]models.TicketComment{},
			}
			router := gin.New()
			handler := NewTicketHandler(repo, nil, nil)
			router.POST("/tickets/:id/comments", withUser(tt.userID, tt.orgID, tt.role), handler.CreateComment)

			req := httptest.NewRequest(http.MethodPost, "/tickets/"+ticketID.String()+"/comments", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			if res.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", res.Code, tt.wantStatus, res.Body.String())
			}
		})
	}
}

func TestListCommentsAccessAndErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgID := uuid.New()
	clientID := uuid.New()
	ticketID := uuid.New()

	t.Run("client lists own ticket comments", func(t *testing.T) {
		repo := &fakeTicketRepo{
			tickets: map[uuid.UUID]*models.Ticket{
				ticketID: {ID: ticketID, OrganizationID: orgID, CreatorID: clientID},
			},
			comments: map[uuid.UUID][]models.TicketComment{
				ticketID: {{ID: uuid.New(), TicketID: ticketID, AuthorID: clientID, Message: "Comment"}},
			},
		}
		router := gin.New()
		handler := NewTicketHandler(repo, nil, nil)
		router.GET("/tickets/:id/comments", withUser(clientID, orgID, models.RoleClient), handler.ListComments)

		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/tickets/"+ticketID.String()+"/comments", nil))

		if res.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
		}
		var comments []ticketCommentResponse
		if err := json.Unmarshal(res.Body.Bytes(), &comments); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if len(comments) != 1 || comments[0].Message != "Comment" {
			t.Fatalf("comments = %+v", comments)
		}
	})

	t.Run("invalid ticket id returns bad request", func(t *testing.T) {
		repo := &fakeTicketRepo{tickets: map[uuid.UUID]*models.Ticket{}, comments: map[uuid.UUID][]models.TicketComment{}}
		router := gin.New()
		handler := NewTicketHandler(repo, nil, nil)
		router.GET("/tickets/:id/comments", withUser(clientID, orgID, models.RoleClient), handler.ListComments)

		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/tickets/not-a-uuid/comments", nil))

		if res.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
		}
	})

	t.Run("missing ticket returns not found", func(t *testing.T) {
		repo := &fakeTicketRepo{tickets: map[uuid.UUID]*models.Ticket{}, comments: map[uuid.UUID][]models.TicketComment{}}
		router := gin.New()
		handler := NewTicketHandler(repo, nil, nil)
		router.GET("/tickets/:id/comments", withUser(clientID, orgID, models.RoleClient), handler.ListComments)

		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/tickets/"+ticketID.String()+"/comments", nil))

		if res.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusNotFound)
		}
	})
}

func TestInvalidStatusTransitionDoesNotUpdateTicket(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	repo := &fakeTicketRepo{
		tickets: map[uuid.UUID]*models.Ticket{
			ticketID: {
				ID:             ticketID,
				OrganizationID: orgID,
				CreatorID:      userID,
				Status:         models.StatusNew,
			},
		},
		comments: map[uuid.UUID][]models.TicketComment{},
	}

	router := gin.New()
	handler := NewTicketHandler(repo, nil, nil)
	router.PATCH("/tickets/:id/status", withUser(userID, orgID, models.RoleOperator), handler.UpdateTicketStatus)

	req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/status", strings.NewReader(`{"status":"Resolved"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d, body = %s", res.Code, http.StatusUnprocessableEntity, res.Body.String())
	}
	if repo.updateCalled {
		t.Fatal("UpdateStatus was called for an invalid transition")
	}
}

func TestAssignTicketAccessRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgID := uuid.New()
	otherOrgID := uuid.New()
	operatorID := uuid.New()
	clientID := uuid.New()
	ticketID := uuid.New()
	engineerID := uuid.New()
	otherOrgEngineerID := uuid.New()
	clientAssigneeID := uuid.New()

	tests := []struct {
		name             string
		userID           uuid.UUID
		orgID            uuid.UUID
		role             models.Role
		body             string
		wantStatus       int
		wantAssignCalled bool
		wantAssigneeID   *uuid.UUID
	}{
		{
			name:             "operator can assign organization engineer",
			userID:           operatorID,
			orgID:            orgID,
			role:             models.RoleOperator,
			body:             `{"assignee_id":"` + engineerID.String() + `"}`,
			wantStatus:       http.StatusOK,
			wantAssignCalled: true,
			wantAssigneeID:   &engineerID,
		},
		{
			name:             "operator can clear assignee",
			userID:           operatorID,
			orgID:            orgID,
			role:             models.RoleOperator,
			body:             `{"assignee_id":null}`,
			wantStatus:       http.StatusOK,
			wantAssignCalled: true,
			wantAssigneeID:   nil,
		},
		{
			name:             "client cannot assign ticket",
			userID:           clientID,
			orgID:            orgID,
			role:             models.RoleClient,
			body:             `{"assignee_id":"` + engineerID.String() + `"}`,
			wantStatus:       http.StatusForbidden,
			wantAssignCalled: false,
		},
		{
			name:             "assignee must belong to organization",
			userID:           operatorID,
			orgID:            orgID,
			role:             models.RoleOperator,
			body:             `{"assignee_id":"` + otherOrgEngineerID.String() + `"}`,
			wantStatus:       http.StatusNotFound,
			wantAssignCalled: false,
		},
		{
			name:             "assignee must be engineer",
			userID:           operatorID,
			orgID:            orgID,
			role:             models.RoleOperator,
			body:             `{"assignee_id":"` + clientAssigneeID.String() + `"}`,
			wantStatus:       http.StatusBadRequest,
			wantAssignCalled: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ticketRepo := &fakeTicketRepo{
				tickets: map[uuid.UUID]*models.Ticket{
					ticketID: {
						ID:             ticketID,
						OrganizationID: orgID,
						CreatorID:      clientID,
						Status:         models.StatusNew,
						Priority:       models.PriorityMedium,
						CreatedAt:      time.Now().UTC(),
					},
				},
				comments: map[uuid.UUID][]models.TicketComment{},
			}
			userRepo := &fakeUserRepo{
				users: map[uuid.UUID]*models.User{
					engineerID: {
						ID:             engineerID,
						OrganizationID: orgID,
						Role:           models.RoleEngineer,
						Email:          "engineer@example.com",
						FirstName:      "Test",
						LastName:       "Engineer",
					},
					otherOrgEngineerID: {
						ID:             otherOrgEngineerID,
						OrganizationID: otherOrgID,
						Role:           models.RoleEngineer,
						Email:          "other@example.com",
						FirstName:      "Other",
						LastName:       "Engineer",
					},
					clientAssigneeID: {
						ID:             clientAssigneeID,
						OrganizationID: orgID,
						Role:           models.RoleClient,
						Email:          "client@example.com",
						FirstName:      "Test",
						LastName:       "Client",
					},
				},
			}
			router := gin.New()
			handler := NewTicketHandler(ticketRepo, userRepo, nil)
			router.PATCH("/tickets/:id/assignee", withUser(tt.userID, tt.orgID, tt.role), handler.AssignTicket)

			req := httptest.NewRequest(http.MethodPatch, "/tickets/"+ticketID.String()+"/assignee", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			res := httptest.NewRecorder()

			router.ServeHTTP(res, req)

			if res.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", res.Code, tt.wantStatus, res.Body.String())
			}
			if ticketRepo.assignCalled != tt.wantAssignCalled {
				t.Fatalf("assignCalled = %v, want %v", ticketRepo.assignCalled, tt.wantAssignCalled)
			}
			if tt.wantAssigneeID == nil {
				if ticketRepo.assignedID != nil {
					t.Fatalf("assignedID = %v, want nil", ticketRepo.assignedID)
				}
				return
			}
			if ticketRepo.assignedID == nil || *ticketRepo.assignedID != *tt.wantAssigneeID {
				t.Fatalf("assignedID = %v, want %v", ticketRepo.assignedID, *tt.wantAssigneeID)
			}
		})
	}
}

func TestAttachmentAccessRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orgID := uuid.New()
	userID := uuid.New()
	ticketID := uuid.New()
	attachmentID := uuid.New()

	t.Run("admin cannot upload attachments", func(t *testing.T) {
		router := gin.New()
		handler := &AttachmentHandler{}
		router.POST("/tickets/:id/attachments", withUser(userID, orgID, models.RoleAdmin), handler.Upload)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		_ = writer.Close()

		req := httptest.NewRequest(http.MethodPost, "/tickets/"+ticketID.String()+"/attachments", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		res := httptest.NewRecorder()

		router.ServeHTTP(res, req)

		if res.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
		}
	})

	t.Run("admin cannot download attachments", func(t *testing.T) {
		router := gin.New()
		handler := &AttachmentHandler{}
		router.GET("/tickets/:id/attachments/:aid/download", withUser(userID, orgID, models.RoleAdmin), handler.Download)

		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/tickets/"+ticketID.String()+"/attachments/"+attachmentID.String()+"/download", nil))

		if res.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
		}
	})

	t.Run("download rejects attachment from another ticket", func(t *testing.T) {
		otherTicketID := uuid.New()
		store := &fakeStorage{}
		handler := &AttachmentHandler{
			ticketRepo: &fakeTicketRepo{
				tickets: map[uuid.UUID]*models.Ticket{
					ticketID: {ID: ticketID, OrganizationID: orgID, CreatorID: userID},
				},
				comments: map[uuid.UUID][]models.TicketComment{},
			},
			attachmentRepo: &fakeAttachmentRepo{
				attachment: &models.Attachment{
					ID:          attachmentID,
					TicketID:    otherTicketID,
					FileName:    "log.txt",
					FilePath:    "uploads/log.txt",
					FileSize:    4,
					ContentType: "text/plain",
				},
			},
			storage: store,
		}
		router := gin.New()
		router.GET("/tickets/:id/attachments/:aid/download", withUser(userID, orgID, models.RoleClient), handler.Download)

		res := httptest.NewRecorder()
		router.ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/tickets/"+ticketID.String()+"/attachments/"+attachmentID.String()+"/download", nil))

		if res.Code != http.StatusForbidden {
			t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
		}
		if store.opened {
			t.Fatal("storage.Open was called for an unauthorized attachment")
		}
	})
}

func withUser(userID, orgID uuid.UUID, role models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("user_id", userID)
		c.Set("organization_id", orgID)
		c.Set("role", role)
		c.Next()
	}
}

var _ repository.TicketRepository = (*fakeTicketRepo)(nil)
var _ repository.UserRepository = (*fakeUserRepo)(nil)
