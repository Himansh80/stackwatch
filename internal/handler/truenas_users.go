package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/stackwatch/platform/internal/client/truenas"
)

// --- T2.9 — Users / Groups / ACL ---

func ListUsers(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r struct {
			HostID string `json:"host_id"`
		}
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		out, err := cli.ListUsers()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"users": out, "total": len(out)})
	}
}

func GetUser(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID     int64  `json:"id"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		u, err := cli.GetUser(r.ID)
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, u)
	}
}

type userReq struct {
	HostID      string `json:"host_id"`
	Username    string `json:"username"`
	FullName    string `json:"full_name,omitempty"`
	Email       string `json:"email,omitempty"`
	Password    string `json:"password,omitempty"`
	UID         int    `json:"uid,omitempty"`
	Home        string `json:"home,omitempty"`
	Shell       string `json:"shell,omitempty"`
	Group       int    `json:"group,omitempty"`
	SSHPassword bool   `json:"ssh_password_enabled,omitempty"`
	Locked      bool   `json:"locked,omitempty"`
}

func userOpts(r userReq) truenas.UserCreate {
	return truenas.UserCreate{
		Username: r.Username, FullName: r.FullName, Email: r.Email,
		Password: r.Password, UID: r.UID, Home: r.Home, Shell: r.Shell,
		Group: r.Group, SSHPassword: r.SSHPassword, Locked: r.Locked,
	}
}

func CreateUser(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r userReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreateUser(userOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func UpdateUser(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID     int64  `json:"id"`
		userReq
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.UpdateUser(r.ID, userOpts(r.userReq)); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"updated": true})
	}
}

func DeleteUser(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID     int64  `json:"id"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.DeleteUser(r.ID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

func ListGroups(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r struct {
			HostID string `json:"host_id"`
		}
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		out, err := cli.ListGroups()
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"groups": out, "total": len(out)})
	}
}

type groupCreateReq struct {
	HostID string `json:"host_id"`
	Name   string `json:"name"`
	GID    int    `json:"gid,omitempty"`
	Perm   string `json:"permissions,omitempty"`
}

func groupOpts(r groupCreateReq) truenas.GroupCreate {
	return truenas.GroupCreate{Name: r.Name, GID: r.GID, Perm: r.Perm}
}

func CreateGroup(store *hostStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r groupCreateReq
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		id, err := cli.CreateGroup(groupOpts(r))
		if err != nil {
			Err(c, err)
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id})
	}
}

func DeleteGroup(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID string `json:"host_id"`
		ID     int64  `json:"id"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.DeleteGroup(r.ID); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"deleted": true})
	}
}

func GetACL(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID     string `json:"host_id"`
		Path       string `json:"path"`
		Simplified bool   `json:"simplified"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		acl, err := cli.GetACL(r.Path, r.Simplified)
		if err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"entries": acl})
	}
}

func SetACL(store *hostStore) gin.HandlerFunc {
	type req struct {
		HostID    string           `json:"host_id"`
		Path      string           `json:"path"`
		Recursive bool             `json:"recursive"`
		ACLType   string           `json:"acltype"`
		Entries   []map[string]any `json:"entries"`
	}
	return func(c *gin.Context) {
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			Bad(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		cli, _, ok := ResolveCreds(c, store, r.HostID, nil)
		if !ok {
			return
		}
		if err := cli.SetACL(truenas.SetACLOptions{
			Path:    r.Path,
			Entries: r.Entries,
			Recurse: r.Recursive,
		}); err != nil {
			Err(c, err)
			return
		}
		OK(c, gin.H{"acl_set": true})
	}
}
