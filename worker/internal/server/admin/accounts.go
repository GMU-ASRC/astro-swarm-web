package admin

import (
	"net/http"
	"strings"

	"astroswarm/worker/internal/server/auth"
	"astroswarm/worker/internal/server/httpx"
	"astroswarm/worker/internal/server/pytime"
	"astroswarm/worker/internal/server/values"
)

const minimumPasswordLength = 8

func textField(data map[string]any, key string) string {
	value, found := data[key]
	if !found {
		return ""
	}
	return values.ToText(value)
}

func plainError(w http.ResponseWriter, status int, message string) error {
	httpx.WriteJSON(w, status, map[string]any{"error": message})
	return nil
}

func (api *API) login(w http.ResponseWriter, r *http.Request) error {
	data := httpx.ReadJSONObjectOrEmpty(r)
	username := strings.TrimSpace(textField(data, "username"))
	password := textField(data, "password")
	if username == "" || password == "" {
		return httpx.BadRequest("username and password are required")
	}
	ctx := r.Context()
	user, err := api.deps.Auth.UserByUsername(ctx, username)
	if err != nil {
		return err
	}
	if user == nil || !auth.CheckPassword(user.PasswordHash, password) {
		return plainError(w, http.StatusUnauthorized, "Invalid username or password")
	}
	session, err := api.deps.Auth.CreateSession(ctx, user)
	if err != nil {
		return err
	}
	refreshed, err := api.deps.Auth.UserByID(ctx, user.ID)
	if err != nil {
		return err
	}
	if refreshed != nil {
		user = refreshed
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"token": session.Token, "expires_at": pytime.Isoformat(session.ExpiresAt), "user": user.ToMap(),
	})
	return nil
}

func (api *API) logout(w http.ResponseWriter, r *http.Request) error {
	if err := api.deps.Auth.RevokeToken(r.Context(), auth.ExtractToken(r)); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	return nil
}

func (api *API) me(w http.ResponseWriter, r *http.Request) error {
	admin, err := api.deps.Auth.CurrentAdmin(r.Context(), r)
	if err != nil {
		return err
	}
	if admin == nil {
		return plainError(w, http.StatusUnauthorized, "Not authenticated")
	}
	if admin.Master {
		httpx.WriteJSON(w, http.StatusOK, map[string]any{"username": "master", "master": true})
		return nil
	}
	output := admin.User.ToMap()
	output["master"] = false
	httpx.WriteJSON(w, http.StatusOK, output)
	return nil
}

func (api *API) changePassword(w http.ResponseWriter, r *http.Request) error {
	admin, err := api.deps.Auth.RequireAdmin(r)
	if err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	newPassword := textField(data, "new_password")
	if values.Length(newPassword) < minimumPasswordLength {
		return httpx.BadRequest("new_password must be at least 8 characters")
	}
	if admin.Master {
		return httpx.Forbidden("The master key login cannot change a user password")
	}
	if !auth.CheckPassword(admin.User.PasswordHash, textField(data, "current_password")) {
		return plainError(w, http.StatusUnauthorized, "Current password is incorrect")
	}
	if err := api.deps.Auth.SetPassword(r.Context(), admin.User.ID, newPassword); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	return nil
}

func (api *API) listUsers(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	rows, err := api.deps.Store.Pool.Query(r.Context(),
		"SELECT id, username, password_hash, created_at, last_login FROM admin_users ORDER BY username ASC")
	if err != nil {
		return err
	}
	defer rows.Close()
	users := []any{}
	for rows.Next() {
		user := &auth.AdminUser{}
		if err := rows.Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt, &user.LastLogin); err != nil {
			return err
		}
		users = append(users, user.ToMap())
	}
	if err := rows.Err(); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, users)
	return nil
}

func (api *API) createUser(w http.ResponseWriter, r *http.Request) error {
	if _, err := api.deps.Auth.RequireAdmin(r); err != nil {
		return err
	}
	data := httpx.ReadJSONObjectOrEmpty(r)
	username := strings.TrimSpace(textField(data, "username"))
	password := textField(data, "password")
	if username == "" {
		return httpx.BadRequest("username is required")
	}
	if values.Length(password) < minimumPasswordLength {
		return httpx.BadRequest("password must be at least 8 characters")
	}
	ctx := r.Context()
	existing, err := api.deps.Auth.UserByUsername(ctx, username)
	if err != nil {
		return err
	}
	if existing != nil {
		return httpx.Conflict("That username already exists")
	}
	user, err := api.deps.Auth.CreateUser(ctx, username, password)
	if err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusCreated, user.ToMap())
	return nil
}

func (api *API) deleteUser(w http.ResponseWriter, r *http.Request) error {
	admin, err := api.deps.Auth.RequireAdmin(r)
	if err != nil {
		return err
	}
	ctx := r.Context()
	target, err := api.deps.Auth.UserByID(ctx, r.PathValue("user_id"))
	if err != nil {
		return err
	}
	if target == nil {
		return httpx.NotFound("User not found")
	}
	var userCount int
	if err := api.deps.Store.Pool.QueryRow(ctx, "SELECT count(*) FROM admin_users").Scan(&userCount); err != nil {
		return err
	}
	if userCount <= 1 {
		return httpx.BadRequest("Cannot delete the last admin user")
	}
	if !admin.Master && admin.User.ID == target.ID {
		return httpx.BadRequest("You cannot delete the account you are signed in with")
	}
	transaction, err := api.deps.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer transaction.Rollback(ctx)
	if _, err := transaction.Exec(ctx, "DELETE FROM admin_sessions WHERE user_id = $1", target.ID); err != nil {
		return err
	}
	if _, err := transaction.Exec(ctx, "DELETE FROM admin_users WHERE id = $1", target.ID); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return err
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
	return nil
}
