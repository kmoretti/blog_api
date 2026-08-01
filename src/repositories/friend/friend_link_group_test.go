package friendsRepositories

import (
	"testing"

	"blog_api/src/model"
	"blog_api/src/testutil"
)

func TestEnsureDefaultFriendLinkGroupUsesDescriptionAndBackfillsEmptyDescription(t *testing.T) {
	db := testutil.NewTestDB(t)

	id, err := EnsureDefaultFriendLinkGroup(db)
	if err != nil {
		t.Fatal(err)
	}
	var group model.FriendLinkGroup
	if err := db.First(&group, id).Error; err != nil {
		t.Fatal(err)
	}
	if group.Description != "我的网上邻居~" {
		t.Fatalf("expected default description, got %q", group.Description)
	}

	if err := db.Model(&group).Update("description", "").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := EnsureDefaultFriendLinkGroup(db); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&group, id).Error; err != nil {
		t.Fatal(err)
	}
	if group.Description != "我的网上邻居~" {
		t.Fatalf("expected empty description to be backfilled, got %q", group.Description)
	}
}

func TestEnsureDefaultFriendLinkGroupPreservesCustomDescription(t *testing.T) {
	db := testutil.NewTestDB(t)
	group := model.FriendLinkGroup{Name: defaultGroupName, Description: "自定义描述"}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}

	if _, err := EnsureDefaultFriendLinkGroup(db); err != nil {
		t.Fatal(err)
	}
	if err := db.First(&group, group.ID).Error; err != nil {
		t.Fatal(err)
	}
	if group.Description != "自定义描述" {
		t.Fatalf("expected custom description to be preserved, got %q", group.Description)
	}
}
