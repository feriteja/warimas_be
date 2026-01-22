package graph

import (
	"fmt"
	"time"
	"warimas-be/internal/graph/model"
	"warimas-be/internal/user"
	"warimas-be/internal/utils"
)

func mapProfileToGraphQL(profile *user.Profile) *model.Profile {
	var dob *string
	if profile.DateOfBirth != nil {
		d := profile.DateOfBirth.Format("2006-01-02")
		dob = &d
	}

	return &model.Profile{
		ID:          profile.ID.String(),
		UserID:      fmt.Sprint(profile.UserID),
		FullName:    profile.FullName,
		Bio:         profile.Bio,
		AvatarURL:   profile.AvatarURL,
		Phone:       profile.Phone,
		Email:       profile.Email,
		DateOfBirth: dob,
		CreatedAt:   utils.StrPtr(profile.CreatedAt.Format(time.RFC3339)),
		UpdatedAt:   utils.StrPtr(profile.UpdatedAt.Format(time.RFC3339)),
	}
}
