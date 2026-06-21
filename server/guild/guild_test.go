package guild

import (
	"testing"
)

// --- CreateGuild ---

func TestCreateGuildFounderGetsFeatherSack(t *testing.T) {
	r := New()
	g, sack, err := r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	if err != nil {
		t.Fatalf("CreateGuild: %v", err)
	}
	if sack.Kind != KindFeatherSack {
		t.Errorf("founder should receive Feather Sack, got kind=%d", sack.Kind)
	}
	if sack.OwnerID != "founder" {
		t.Errorf("sack owner: got %q, want founder", sack.OwnerID)
	}
	if g.FounderID != "founder" {
		t.Errorf("guild founder: got %q", g.FounderID)
	}
}

func TestCreateGuildFounderIsLeader(t *testing.T) {
	r := New()
	_, _, _ = r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	g, _ := r.GetGuild("g1")
	members := g.Members()
	if len(members) != 1 {
		t.Fatalf("want 1 member, got %d", len(members))
	}
	if members[0].Role != RoleLeader {
		t.Errorf("founder role: got %d, want %d (Leader)", members[0].Role, RoleLeader)
	}
}

func TestCreateGuildCanChat(t *testing.T) {
	r := New()
	_, _, _ = r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	if !r.CanChat("g1", "founder") {
		t.Error("founder should be able to chat immediately after CreateGuild")
	}
}

// --- ForgeFeather (members join via item) ---

func TestForgeFeatherAddsNewMember(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	feather, err := r.ForgeFeather("g1", "founder", "alice")
	if err != nil {
		t.Fatalf("ForgeFeather: %v", err)
	}
	if feather.Kind != KindFeather {
		t.Errorf("should be Feather, got kind=%d", feather.Kind)
	}
	if feather.OwnerID != "alice" {
		t.Errorf("owner: got %q, want alice", feather.OwnerID)
	}
	if !r.CanChat("g1", "alice") {
		t.Error("alice should be able to chat after receiving Feather")
	}
}

func TestForgeFeatherNonMemberCannotIssue(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	_, err := r.ForgeFeather("g1", "outsider", "bob")
	if err != ErrNotMember {
		t.Errorf("outsider issuing Feather: want ErrNotMember, got %v", err)
	}
}

func TestForgeFeatherMemberCannotIssue(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	r.ForgeFeather("g1", "founder", "alice") // alice = Member role
	_, err := r.ForgeFeather("g1", "alice", "bob")
	if err != ErrInsufficientRole {
		t.Errorf("member issuing Feather: want ErrInsufficientRole, got %v", err)
	}
}

func TestForgeFeatherDuplicateReturnsError(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	r.ForgeFeather("g1", "founder", "alice")
	_, err := r.ForgeFeather("g1", "founder", "alice")
	if err != ErrAlreadyMember {
		t.Errorf("duplicate Feather: want ErrAlreadyMember, got %v", err)
	}
}

// --- ForgeFeatherSack (officer promotion) ---

func TestForgeFeatherSackPromotesToOfficer(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	r.ForgeFeather("g1", "founder", "alice")
	sack, err := r.ForgeFeatherSack("g1", "founder", "alice")
	if err != nil {
		t.Fatalf("ForgeFeatherSack: %v", err)
	}
	if sack.Kind != KindFeatherSack {
		t.Errorf("promoted item should be Feather Sack, got kind=%d", sack.Kind)
	}
	g, _ := r.GetGuild("g1")
	for _, m := range g.Members() {
		if m.CharacterID == "alice" && m.Role != RoleOfficer {
			t.Errorf("alice should be Officer after ForgeFeatherSack, got role=%d", m.Role)
		}
	}
}

func TestForgeFeatherSackOfficerCannotPromote(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	r.ForgeFeather("g1", "founder", "alice")
	r.ForgeFeatherSack("g1", "founder", "alice") // alice = Officer
	r.ForgeFeather("g1", "alice", "bob")         // alice can forge Feathers as Officer
	_, err := r.ForgeFeatherSack("g1", "alice", "bob")
	if err != ErrInsufficientRole {
		t.Errorf("Officer forging Sack: want ErrInsufficientRole, got %v", err)
	}
}

func TestForgeFeatherSackNonMemberCannotPromote(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	_, err := r.ForgeFeatherSack("g1", "outsider", "founder")
	if err != ErrNotMember {
		t.Errorf("outsider forging Sack: want ErrNotMember, got %v", err)
	}
}

// --- RevokeItem ---

func TestRevokeFeatherRemovesMember(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	feather, _ := r.ForgeFeather("g1", "founder", "alice")
	if err := r.RevokeItem("g1", "founder", feather.ID); err != nil {
		t.Fatalf("RevokeItem: %v", err)
	}
	if r.CanChat("g1", "alice") {
		t.Error("alice should NOT be able to chat after Feather revoked")
	}
}

func TestRevokeItemAlreadyRevokedErrors(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	feather, _ := r.ForgeFeather("g1", "founder", "alice")
	r.RevokeItem("g1", "founder", feather.ID)
	err := r.RevokeItem("g1", "founder", feather.ID)
	if err != ErrItemRevoked {
		t.Errorf("double revoke: want ErrItemRevoked, got %v", err)
	}
}

func TestRevokeOwnItemErrors(t *testing.T) {
	r := New()
	_, sack, _ := r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	err := r.RevokeItem("g1", "founder", sack.ID)
	if err == nil {
		t.Error("revoking own item should return error")
	}
}

func TestOfficerCannotRevokeSack(t *testing.T) {
	r := New()
	_, sack, _ := r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	r.ForgeFeather("g1", "founder", "alice")
	r.ForgeFeatherSack("g1", "founder", "alice") // alice = Officer
	// alice tries to revoke founder's sack
	err := r.RevokeItem("g1", "alice", sack.ID)
	if err != ErrInsufficientRole {
		t.Errorf("Officer revoking Sack: want ErrInsufficientRole, got %v", err)
	}
}

func TestMemberCannotRevoke(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	r.ForgeFeather("g1", "founder", "alice")
	feather2, _ := r.ForgeFeather("g1", "founder", "bob")
	err := r.RevokeItem("g1", "alice", feather2.ID) // alice = Member
	if err != ErrInsufficientRole {
		t.Errorf("Member revoking: want ErrInsufficientRole, got %v", err)
	}
}

// --- CanChat / GuildID ---

func TestCanChatFalseForNonMember(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	if r.CanChat("g1", "outsider") {
		t.Error("outsider should not be able to chat")
	}
}

func TestGuildIDReturnsEmptyForNonMember(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	if id := r.GuildID("outsider"); id != "" {
		t.Errorf("GuildID for non-member: got %q, want empty", id)
	}
}

func TestGuildIDReturnsCorrectGuild(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	r.ForgeFeather("g1", "founder", "alice")
	if id := r.GuildID("alice"); id != "g1" {
		t.Errorf("GuildID: got %q, want g1", id)
	}
}

// --- Items ---

func TestItemIsActiveAfterForge(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	feather, _ := r.ForgeFeather("g1", "founder", "alice")
	if !feather.IsActive() {
		t.Error("freshly forged Feather should be active")
	}
}

func TestItemNotActiveAfterRevoke(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	feather, _ := r.ForgeFeather("g1", "founder", "alice")
	r.RevokeItem("g1", "founder", feather.ID)
	g, _ := r.GetGuild("g1")
	for _, it := range g.Items() {
		if it.ID == feather.ID && it.IsActive() {
			t.Error("revoked Feather should not be active")
		}
	}
}

func TestOfficerCanForgeFeathers(t *testing.T) {
	r := New()
	r.CreateGuild("g1", "DragonOrder", "[DO]", "founder")
	r.ForgeFeather("g1", "founder", "alice")
	r.ForgeFeatherSack("g1", "founder", "alice") // alice = Officer
	_, err := r.ForgeFeather("g1", "alice", "charlie")
	if err != nil {
		t.Errorf("Officer should be able to forge Feathers: %v", err)
	}
	if !r.CanChat("g1", "charlie") {
		t.Error("charlie should be able to chat after receiving Feather from Officer")
	}
}

func TestUnknownGuildErrors(t *testing.T) {
	r := New()
	_, err := r.GetGuild("nonexistent")
	if err != ErrGuildNotFound {
		t.Errorf("unknown guild: want ErrGuildNotFound, got %v", err)
	}
}
