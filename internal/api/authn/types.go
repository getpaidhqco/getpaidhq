package authn

// UserRole represents the role of a user
type UserRole string

const (
	Admin   UserRole = "admin"
	Support UserRole = "support"
	Owner   UserRole = "owner"
	Member  UserRole = "member"
)

type User struct {
	OrgId       string     `json:"org_id"`
	Id          string     `json:"id"`
	Email       string     `json:"email"`
	PrimaryRole UserRole   `json:"primary_role"`
	Roles       []UserRole `json:"roles"`
}

func NewUser(orgId, id, email string, roles []UserRole) User {
	return User{
		OrgId:       orgId,
		Id:          id,
		Email:       email,
		PrimaryRole: GetPrimaryRole(roles),
		Roles:       roles,
	}
}

func GetPrimaryRole(roles []UserRole) UserRole {
	rolesRank := map[UserRole]int{
		Admin:   4,
		Support: 3,
		Owner:   2,
		Member:  1,
	}

	primaryRole := Member
	for _, role := range roles {
		if rolesRank[role] > rolesRank[primaryRole] {
			primaryRole = role
		}
	}
	return primaryRole
}
