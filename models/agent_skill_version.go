package models

type AgentSkillVersion struct {
	ID                 int64  `orm:"column(id);auto" json:"id"`
	SkillID            int64  `orm:"column(skill_id);index" json:"skill_id"`
	Name               string `orm:"column(name);size(64);index" json:"name"`
	PackageHash        string `orm:"column(package_hash);size(64);unique" json:"package_hash"`
	Version            string `orm:"column(version);size(64)" json:"version"`
	Description        string `orm:"column(description);type(text)" json:"description"`
	License            string `orm:"column(license);size(512)" json:"license"`
	Compatibility      string `orm:"column(compatibility);size(500)" json:"compatibility"`
	MetadataJSON       string `orm:"column(metadata_json);type(text)" json:"metadata_json"`
	RequestedToolsJSON string `orm:"column(requested_tools_json);type(text)" json:"requested_tools_json"`
	ValidationStatus   string `orm:"column(validation_status);size(32)" json:"validation_status"`
	ValidationJSON     string `orm:"column(validation_json);type(text)" json:"validation_json"`
	Source             string `orm:"column(source);size(32)" json:"source"`
	SourceRef          string `orm:"column(source_ref);type(text)" json:"source_ref"`
	PackagePath        string `orm:"column(package_path);type(text)" json:"package_path"`
	FileCount          int    `orm:"column(file_count);default(0)" json:"file_count"`
	TotalBytes         int64  `orm:"column(total_bytes);default(0)" json:"total_bytes"`
	CreatedAt          int64  `orm:"column(created_at);index" json:"created_at"`
}

func (*AgentSkillVersion) TableName() string { return "agent_skill_versions" }

type AgentSkillPermission struct {
	ID            int64  `orm:"column(id);auto" json:"id"`
	SkillID       int64  `orm:"column(skill_id);index" json:"skill_id"`
	VersionID     int64  `orm:"column(version_id);index" json:"version_id"`
	RequestedName string `orm:"column(requested_name);size(255)" json:"requested_name"`
	ResolvedName  string `orm:"column(resolved_name);size(255);index" json:"resolved_name"`
	Risk          string `orm:"column(risk);size(16)" json:"risk"`
	Status        string `orm:"column(status);size(32)" json:"status"`
	Granted       int    `orm:"column(granted);default(0)" json:"granted"`
	Reason        string `orm:"column(reason);type(text)" json:"reason"`
	CreatedAt     int64  `orm:"column(created_at)" json:"created_at"`
	UpdatedAt     int64  `orm:"column(updated_at)" json:"updated_at"`
}

func (*AgentSkillPermission) TableName() string { return "agent_skill_permissions" }
func (*AgentSkillPermission) TableUnique() [][]string {
	return [][]string{{"VersionID", "RequestedName"}}
}
