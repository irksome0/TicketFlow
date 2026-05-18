// internal/database/database.go
package database

import (
	"fmt"
	"log"
	"strings"

	"ticketflow-api/internal/config"
	"ticketflow-api/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDB(cfg *config.Config) *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		cfg.DBHost,
		cfg.DBPort,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBSSLMode,
		cfg.DBTimeZone,
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Помилка підключення до бази даних: %v", err)
	}

	bootstrapPostgres(database)

	if err := database.AutoMigrate(
		&models.Organization{},
		&models.User{},
		&models.OrganizationInvite{},
		&models.Ticket{},
		&models.TicketStatusHistory{},
		&models.Attachment{},
		&models.TicketComment{},
	); err != nil {
		log.Fatalf("Помилка міграції бази даних: %v", err)
	}

	if cfg.DBRLS {
		configureTenantRLS(database)
	}

	log.Println("Підключення до бази даних встановлено успішно")
	return database
}

func bootstrapPostgres(database *gorm.DB) {
	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS pgcrypto`,
		createEnumSQL("user_role", []string{"client", "operator", "engineer", "admin"}),
		createEnumSQL("ticket_priority", []string{"High", "Medium", "Low"}),
		createEnumSQL("ticket_status", []string{
			"New",
			"In Progress",
			"Pending",
			"Waiting for Customer",
			"On Hold",
			"Resolved",
			"Closed",
			"Reopened",
		}),
	}

	for _, statement := range statements {
		if err := database.Exec(statement).Error; err != nil {
			log.Fatalf("Помилка підготовки PostgreSQL схеми: %v", err)
		}
	}

	ensureEnumValues(database, "ticket_status", []string{
		"Pending",
		"Waiting for Customer",
		"On Hold",
	})
}

func createEnumSQL(name string, values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}

	return fmt.Sprintf(`
DO $$
BEGIN
	IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = '%s') THEN
		CREATE TYPE %s AS ENUM (%s);
	END IF;
END
$$;`, name, name, strings.Join(quoted, ", "))
}

func ensureEnumValues(database *gorm.DB, name string, values []string) {
	for _, value := range values {
		statement := fmt.Sprintf(
			"ALTER TYPE %s ADD VALUE IF NOT EXISTS '%s'",
			name,
			strings.ReplaceAll(value, "'", "''"),
		)
		if err := database.Exec(statement).Error; err != nil {
			log.Printf("Не вдалося оновити enum %s значенням %q: %v", name, value, err)
		}
	}
}

func configureTenantRLS(database *gorm.DB) {
	functions := []string{
		`CREATE OR REPLACE FUNCTION ticketflow_current_organization_id()
			RETURNS uuid
			LANGUAGE sql
			STABLE
			AS $$
				SELECT NULLIF(current_setting('ticketflow.organization_id', true), '')::uuid
			$$`,
		`CREATE OR REPLACE FUNCTION ticketflow_rls_bypass()
			RETURNS boolean
			LANGUAGE sql
			STABLE
			AS $$
				SELECT COALESCE(current_setting('ticketflow.rls_bypass', true), '') = 'on'
			$$`,
	}

	statements := []string{
		`ALTER TABLE organizations ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE users ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE organization_invites ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE tickets ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE attachments ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE ticket_comments ENABLE ROW LEVEL SECURITY`,
		`ALTER TABLE ticket_status_histories ENABLE ROW LEVEL SECURITY`,

		dropPolicySQL("organizations", "tenant_organizations_isolation"),
		dropPolicySQL("users", "tenant_users_isolation"),
		dropPolicySQL("organization_invites", "tenant_invites_isolation"),
		dropPolicySQL("tickets", "tenant_tickets_isolation"),
		dropPolicySQL("attachments", "tenant_attachments_isolation"),
		dropPolicySQL("ticket_comments", "tenant_comments_isolation"),
		dropPolicySQL("ticket_status_histories", "tenant_status_history_isolation"),

		`CREATE POLICY tenant_organizations_isolation ON organizations
			USING (ticketflow_rls_bypass() OR id = ticketflow_current_organization_id())
			WITH CHECK (ticketflow_rls_bypass() OR id = ticketflow_current_organization_id())`,
		`CREATE POLICY tenant_users_isolation ON users
			USING (ticketflow_rls_bypass() OR organization_id = ticketflow_current_organization_id())
			WITH CHECK (ticketflow_rls_bypass() OR organization_id = ticketflow_current_organization_id())`,
		`CREATE POLICY tenant_invites_isolation ON organization_invites
			USING (ticketflow_rls_bypass() OR organization_id = ticketflow_current_organization_id())
			WITH CHECK (ticketflow_rls_bypass() OR organization_id = ticketflow_current_organization_id())`,
		`CREATE POLICY tenant_tickets_isolation ON tickets
			USING (ticketflow_rls_bypass() OR organization_id = ticketflow_current_organization_id())
			WITH CHECK (ticketflow_rls_bypass() OR organization_id = ticketflow_current_organization_id())`,
		`CREATE POLICY tenant_attachments_isolation ON attachments
			USING (
				ticketflow_rls_bypass()
				OR EXISTS (
					SELECT 1 FROM tickets
					WHERE tickets.id = attachments.ticket_id
					AND tickets.organization_id = ticketflow_current_organization_id()
				)
			)
			WITH CHECK (
				ticketflow_rls_bypass()
				OR EXISTS (
					SELECT 1 FROM tickets
					WHERE tickets.id = attachments.ticket_id
					AND tickets.organization_id = ticketflow_current_organization_id()
				)
			)`,
		`CREATE POLICY tenant_comments_isolation ON ticket_comments
			USING (
				ticketflow_rls_bypass()
				OR EXISTS (
					SELECT 1 FROM tickets
					WHERE tickets.id = ticket_comments.ticket_id
					AND tickets.organization_id = ticketflow_current_organization_id()
				)
			)
			WITH CHECK (
				ticketflow_rls_bypass()
				OR EXISTS (
					SELECT 1 FROM tickets
					WHERE tickets.id = ticket_comments.ticket_id
					AND tickets.organization_id = ticketflow_current_organization_id()
				)
			)`,
		`CREATE POLICY tenant_status_history_isolation ON ticket_status_histories
			USING (
				ticketflow_rls_bypass()
				OR EXISTS (
					SELECT 1 FROM tickets
					WHERE tickets.id = ticket_status_histories.ticket_id
					AND tickets.organization_id = ticketflow_current_organization_id()
				)
			)
			WITH CHECK (
				ticketflow_rls_bypass()
				OR EXISTS (
					SELECT 1 FROM tickets
					WHERE tickets.id = ticket_status_histories.ticket_id
					AND tickets.organization_id = ticketflow_current_organization_id()
				)
			)`,
	}

	for _, statement := range functions {
		if err := database.Exec(statement).Error; err != nil {
			log.Fatalf("Помилка створення RLS-функції: %v", err)
		}
	}
	for _, statement := range statements {
		if err := database.Exec(statement).Error; err != nil {
			log.Fatalf("Помилка налаштування RLS: %v", err)
		}
	}
}

func dropPolicySQL(tableName, policyName string) string {
	return fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s", policyName, tableName)
}
