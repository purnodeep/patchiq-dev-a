package compliance

const (
	FrameworkNIST80053 = "NIST-800-53"
	FrameworkPCIDSSv4  = "PCI-DSS"
	FrameworkCIS       = "CIS"
	FrameworkHIPAA     = "HIPAA"
	FrameworkISO27001  = "ISO-27001"
	FrameworkSOC2      = "SOC-2"
)

// SeverityTier defines an SLA timeline for a severity level.
type SeverityTier struct {
	Label   string
	Days    *int
	CVSSMin float64
	CVSSMax float64
}

// Control represents a compliance control within a framework.
type Control struct {
	ID              string
	Name            string
	Description     string
	Category        string
	RemediationHint string
	SLATiers        []SeverityTier
	CheckType       string // "asset_inventory", "software_inventory", "vuln_scanning", etc. Empty means SLA.
	CheckConfig     []byte // JSONB check configuration for custom controls (nil for built-in)
}

// SLADays returns the SLA days for a given severity label, or nil if not found.
func (c *Control) SLADays(severity string) *int {
	for _, tier := range c.SLATiers {
		if tier.Label == severity {
			return tier.Days
		}
	}
	return nil
}

// SLADaysByCVSS returns the SLA days and severity label for a given CVSS score.
func (c *Control) SLADaysByCVSS(cvss float64) (*int, string) {
	for _, tier := range c.SLATiers {
		if cvss >= tier.CVSSMin && cvss <= tier.CVSSMax {
			return tier.Days, tier.Label
		}
	}
	return nil, ""
}

// Framework represents a compliance framework with its controls.
type Framework struct {
	ID                   string
	Name                 string
	Version              string
	Description          string
	ApplicableIndustries []string
	Controls             []Control
	DefaultScoringMethod string
}

// GetControl returns the control with the given ID, or nil if not found.
func (f *Framework) GetControl(controlID string) *Control {
	for i := range f.Controls {
		if f.Controls[i].ID == controlID {
			return &f.Controls[i]
		}
	}
	return nil
}

// PatchSLAControl returns the first control that has SLA tiers defined, or nil.
func (f *Framework) PatchSLAControl() *Control {
	for i := range f.Controls {
		if len(f.Controls[i].SLATiers) > 0 {
			return &f.Controls[i]
		}
	}
	return nil
}

var registry = map[string]*Framework{
	FrameworkNIST80053: {
		ID:                   FrameworkNIST80053,
		Name:                 "NIST 800-53 Rev. 5",
		Version:              "5",
		Description:          "Security and Privacy Controls for Information Systems and Organizations",
		ApplicableIndustries: []string{"government", "defense", "healthcare"},
		DefaultScoringMethod: "strictest",
		Controls: []Control{
			{
				ID:              "SI-2",
				Name:            "Flaw Remediation",
				Description:     "Identify, report, and correct system flaws.",
				Category:        "System and Information Integrity",
				RemediationHint: "Deploy patches within SLA timelines using automated patch management tools.",
				SLATiers:        nistPatchSLATiers(),
			},
			{
				ID:              "RA-5",
				Name:            "Vulnerability Monitoring and Scanning",
				Description:     "Verify all endpoints are covered by vulnerability scanning with CVE correlation against the NVD and CISA KEV catalogs.",
				Category:        "Risk Assessment",
				RemediationHint: "Ensure agents are running inventory scans so the server can correlate installed packages against known CVEs.",
			},
			{
				ID:              "CM-3",
				Name:            "Configuration Change Control",
				Description:     "Verify configuration changes (patches) are deployed through controlled change management processes with acceptable success rates.",
				Category:        "Configuration Management",
				RemediationHint: "Route all patch deployments through PatchIQ workflows. Maintain deployment success rate above 80%.",
			},
		},
	},
	FrameworkPCIDSSv4: {
		ID:                   FrameworkPCIDSSv4,
		Name:                 "PCI DSS v4.0",
		Version:              "4.0",
		Description:          "Payment Card Industry Data Security Standard",
		ApplicableIndustries: []string{"finance", "retail", "ecommerce"},
		DefaultScoringMethod: "worst_case",
		Controls: []Control{
			{
				ID:              "6.3.3",
				Name:            "Patch Management",
				Description:     "Install applicable security patches within a defined timeframe.",
				Category:        "Develop and Maintain Secure Systems",
				RemediationHint: "Apply critical and high-severity patches within 30 days of release.",
				SLATiers:        pciPatchSLATiers(),
			},
			{
				ID:              "11.3.1",
				Name:            "Vulnerability Scanning",
				Description:     "Perform internal vulnerability scans on all in-scope system components and verify scan coverage across all endpoints.",
				Category:        "Regularly Test Security Systems",
				RemediationHint: "Ensure all PCI-scoped endpoints have the agent installed and reporting inventory for CVE correlation.",
			},
		},
	},
	FrameworkCIS: {
		ID:                   FrameworkCIS,
		Name:                 "CIS Controls v8",
		Version:              "8",
		Description:          "Center for Internet Security Critical Security Controls",
		ApplicableIndustries: []string{"government", "finance", "healthcare", "technology"},
		DefaultScoringMethod: "weighted",
		Controls: []Control{
			{
				ID:              "CIS-1.1",
				Name:            "Enterprise Asset Inventory",
				Description:     "Verify all enterprise assets are inventoried with hardware details and actively reporting to the management platform.",
				Category:        "IG1",
				RemediationHint: "Ensure all endpoints have the PatchIQ agent installed and reporting hardware inventory within the last 24 hours.",
			},
			{
				ID:              "CIS-2.1",
				Name:            "Software Inventory",
				Description:     "Maintain an accurate software inventory by ensuring all endpoints have completed a package scan within the last 7 days.",
				Category:        "IG1",
				RemediationHint: "Configure agent scan intervals to run at least weekly. Check agent connectivity for endpoints with stale inventory.",
			},
			{
				ID:              "CIS-3.1",
				Name:            "Data Protection",
				Description:     "Establish and maintain a data management process. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "IG1",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "CIS-4.1",
				Name:            "Secure Configuration",
				Description:     "Ensure all patch deployments are executed through the approved change management platform with documented success rates.",
				Category:        "IG1",
				RemediationHint: "Use PatchIQ deployment workflows for all patch operations. Investigate and resolve failed deployments promptly.",
			},
			{
				ID:              "CIS-5.1",
				Name:            "Account Management",
				Description:     "Establish and maintain an inventory of all accounts. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "IG1",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "CIS-6.1",
				Name:            "Access Control Management",
				Description:     "Establish an access granting process based on least privilege. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "IG2",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "CIS-7.1",
				Name:            "Vulnerability Management",
				Description:     "Establish and maintain a vulnerability management process.",
				Category:        "IG2",
				RemediationHint: "Remediate vulnerabilities based on severity within defined SLA timelines.",
				SLATiers:        cisPatchSLATiers(),
			},
			{
				ID:              "CIS-8.1",
				Name:            "Audit Log Management",
				Description:     "Establish and maintain an audit log management process. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "IG2",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "CIS-9.1",
				Name:            "Email and Web Browser Protections",
				Description:     "Ensure only approved browsers and email clients are used. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "IG2",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "CIS-10.1",
				Name:            "Malware Defenses",
				Description:     "Deploy and maintain anti-malware software on all enterprise assets. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "IG3",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
		},
	},
	FrameworkHIPAA: {
		ID:                   FrameworkHIPAA,
		Name:                 "HIPAA Security Rule",
		Version:              "2013",
		Description:          "Health Insurance Portability and Accountability Act Security Standards",
		ApplicableIndustries: []string{"healthcare", "insurance", "pharmaceuticals"},
		DefaultScoringMethod: "strictest",
		Controls: []Control{
			{
				ID:              "HIPAA-164.308a1",
				Name:            "Security Management Process",
				Description:     "Implement policies and procedures to prevent, detect, contain, and correct security violations. Evaluated via CISA KEV exposure and critical vulnerability posture.",
				Category:        "Administrative",
				RemediationHint: "Prioritize remediation of CISA Known Exploited Vulnerabilities. Maintain zero KEV exposure across all endpoints.",
			},
			{
				ID:              "HIPAA-164.308a3",
				Name:            "Workforce Security",
				Description:     "Implement policies to ensure workforce members have appropriate access to ePHI. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Administrative",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "HIPAA-164.308a4",
				Name:            "Information Access Management",
				Description:     "Implement policies for authorizing access to ePHI. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Administrative",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "HIPAA-164.308a5",
				Name:            "Security Awareness and Training",
				Description:     "Implement a security awareness and training program for all workforce members. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Administrative",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "HIPAA-164.310a1",
				Name:            "Facility Access Controls",
				Description:     "Implement policies to limit physical access to electronic information systems. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Physical",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "HIPAA-164.310b",
				Name:            "Workstation Use",
				Description:     "Implement policies that specify proper functions and physical attributes of workstations. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Physical",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "HIPAA-164.312a1",
				Name:            "Access Control",
				Description:     "Implement technical policies to allow access only to authorized persons. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Technical",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "HIPAA-164.312b",
				Name:            "Audit Controls",
				Description:     "Implement hardware, software, and procedural mechanisms to record and examine activity in systems that contain ePHI. Evaluated via agent health monitoring.",
				Category:        "Technical",
				RemediationHint: "Ensure all endpoints maintain active agent connections with heartbeats within the last 24 hours.",
			},
			{
				ID:              "HIPAA-164.312c1",
				Name:            "Integrity",
				Description:     "Implement policies and procedures to protect ePHI from improper alteration or destruction. Evaluated via patch deployment integrity and success rates.",
				Category:        "Technical",
				RemediationHint: "Maintain deployment success rate above 90%. Investigate all failed patch installations promptly.",
			},
			{
				ID:              "HIPAA-164.312d",
				Name:            "Transmission Security",
				Description:     "Implement technical security measures to guard against unauthorized access to ePHI during transmission.",
				Category:        "Technical",
				RemediationHint: "Encrypt all ePHI in transit using TLS 1.2 or higher and patch network devices within SLA.",
				SLATiers:        hipaaPatchSLATiers(),
			},
		},
	},
	FrameworkISO27001: {
		ID:                   FrameworkISO27001,
		Name:                 "ISO 27001:2022",
		Version:              "2022",
		Description:          "Information Security Management Systems Requirements",
		ApplicableIndustries: []string{"technology", "finance", "manufacturing", "consulting"},
		DefaultScoringMethod: "weighted",
		Controls: []Control{
			{
				ID:              "ISO-A5.1",
				Name:            "Policies for Information Security",
				Description:     "A set of policies for information security shall be defined and approved by management. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Organizational",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "ISO-A6.1",
				Name:            "Screening",
				Description:     "Background verification checks on candidates shall be carried out prior to joining. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "People",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "ISO-A7.1",
				Name:            "Physical Security Perimeters",
				Description:     "Security perimeters shall be defined and used to protect areas containing information. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Physical",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "ISO-A8.1",
				Name:            "User Endpoint Devices",
				Description:     "User endpoint devices storing or processing organizational information shall be protected. Evaluated via asset inventory completeness and agent reporting.",
				Category:        "Technological",
				RemediationHint: "Ensure all endpoint devices have the PatchIQ agent enrolled and actively reporting hardware inventory.",
			},
			{
				ID:              "ISO-A8.2",
				Name:            "Privileged Access",
				Description:     "The allocation and use of privileged access rights shall be restricted and managed.",
				Category:        "Technological",
				RemediationHint: "Implement privileged access management and patch privileged systems within SLA.",
				SLATiers:        isoPatchSLATiers(),
			},
			{
				ID:              "ISO-A8.8",
				Name:            "Management of Technical Vulnerabilities",
				Description:     "Information about technical vulnerabilities shall be obtained, exposure evaluated, and appropriate measures taken. Evaluated via critical CVE remediation timeliness.",
				Category:        "Technological",
				RemediationHint: "Remediate critical and high-severity CVEs within 30 days of detection. Prioritize CISA KEV entries.",
			},
			{
				ID:              "ISO-A8.9",
				Name:            "Configuration Management",
				Description:     "Configurations of hardware, software, services, and networks shall be established, documented, and managed. Evaluated via deployment governance.",
				Category:        "Technological",
				RemediationHint: "Deploy all configuration changes through PatchIQ workflows. Maintain deployment success rate above 80%.",
			},
			{
				ID:              "ISO-A8.16",
				Name:            "Monitoring Activities",
				Description:     "Networks, systems, and applications shall be monitored for anomalous behaviour. Evaluated via endpoint agent monitoring coverage.",
				Category:        "Technological",
				RemediationHint: "Ensure 95% or more of endpoints have active agent heartbeats. Investigate offline endpoints promptly.",
			},
		},
	},
	FrameworkSOC2: {
		ID:                   FrameworkSOC2,
		Name:                 "SOC 2 Type II",
		Version:              "2017",
		Description:          "Service Organization Control 2 Trust Services Criteria",
		ApplicableIndustries: []string{"technology", "saas", "cloud", "finance"},
		DefaultScoringMethod: "worst_case",
		Controls: []Control{
			{
				ID:              "SOC2-CC6.1",
				Name:            "Logical and Physical Access Controls",
				Description:     "The entity implements logical and physical access controls to protect against threats. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Security",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "SOC2-CC7.1",
				Name:            "System Operations",
				Description:     "The entity uses detection and monitoring procedures to identify changes to configurations and new vulnerabilities. Evaluated via vulnerability scan coverage.",
				Category:        "Security",
				RemediationHint: "Ensure all endpoints have agents reporting inventory for CVE correlation against NVD and CISA KEV feeds.",
			},
			{
				ID:              "SOC2-CC8.1",
				Name:            "Change Management",
				Description:     "The entity authorizes, designs, develops, configures, documents, tests, approves, and implements changes to infrastructure and software. Evaluated via deployment governance.",
				Category:        "Security",
				RemediationHint: "Route all changes through PatchIQ deployment workflows. Maintain documented success rates above 80%.",
			},
			{
				ID:              "SOC2-A1.1",
				Name:            "System Availability",
				Description:     "The entity maintains, monitors, and evaluates current processing capacity and use of system components to manage capacity demand. Evaluated via endpoint availability monitoring.",
				Category:        "Availability",
				RemediationHint: "Maintain 95% or higher endpoint availability. Investigate and remediate offline agents promptly.",
			},
			{
				ID:              "SOC2-C1.1",
				Name:            "Confidential Information",
				Description:     "The entity identifies and maintains confidential information to meet objectives.",
				Category:        "Confidentiality",
				RemediationHint: "Classify data, encrypt confidential information, and patch systems within SLA.",
				SLATiers:        soc2PatchSLATiers(),
			},
			{
				ID:              "SOC2-PI1.1",
				Name:            "Processing Integrity",
				Description:     "The entity uses processing integrity objectives to ensure system processing is complete and accurate. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Processing Integrity",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "SOC2-P1.1",
				Name:            "Privacy Notice",
				Description:     "The entity provides notice about its privacy practices to data subjects. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Privacy",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
			{
				ID:              "SOC2-P6.1",
				Name:            "Data Quality",
				Description:     "The entity collects and maintains accurate, complete personal information. [Automated evaluation requires agent-side configuration data collection — available in a future release.]",
				Category:        "Privacy",
				RemediationHint: "This control will be automatically evaluated when agent configuration collection modules are enabled.",
			},
		},
	},
}

func nistPatchSLATiers() []SeverityTier {
	return []SeverityTier{
		{Label: "critical", Days: intP(15), CVSSMin: 9.0, CVSSMax: 10.0},
		{Label: "high", Days: intP(30), CVSSMin: 7.0, CVSSMax: 8.9},
		{Label: "moderate", Days: intP(90), CVSSMin: 4.0, CVSSMax: 6.9},
		{Label: "low", Days: nil, CVSSMin: 0.0, CVSSMax: 3.9},
	}
}

func pciPatchSLATiers() []SeverityTier {
	return []SeverityTier{
		{Label: "critical", Days: intP(30), CVSSMin: 9.0, CVSSMax: 10.0},
		{Label: "high", Days: intP(30), CVSSMin: 7.0, CVSSMax: 8.9},
		{Label: "moderate", Days: nil, CVSSMin: 4.0, CVSSMax: 6.9},
		{Label: "low", Days: nil, CVSSMin: 0.0, CVSSMax: 3.9},
	}
}

func cisPatchSLATiers() []SeverityTier {
	return []SeverityTier{
		{Label: "critical", Days: intP(15), CVSSMin: 9.0, CVSSMax: 10.0},
		{Label: "high", Days: intP(30), CVSSMin: 7.0, CVSSMax: 8.9},
		{Label: "moderate", Days: intP(90), CVSSMin: 4.0, CVSSMax: 6.9},
		{Label: "low", Days: nil, CVSSMin: 0.0, CVSSMax: 3.9},
	}
}

func hipaaPatchSLATiers() []SeverityTier {
	return []SeverityTier{
		{Label: "critical", Days: intP(15), CVSSMin: 9.0, CVSSMax: 10.0},
		{Label: "high", Days: intP(30), CVSSMin: 7.0, CVSSMax: 8.9},
		{Label: "moderate", Days: intP(90), CVSSMin: 4.0, CVSSMax: 6.9},
		{Label: "low", Days: nil, CVSSMin: 0.0, CVSSMax: 3.9},
	}
}

func isoPatchSLATiers() []SeverityTier {
	return []SeverityTier{
		{Label: "critical", Days: intP(15), CVSSMin: 9.0, CVSSMax: 10.0},
		{Label: "high", Days: intP(30), CVSSMin: 7.0, CVSSMax: 8.9},
		{Label: "moderate", Days: intP(90), CVSSMin: 4.0, CVSSMax: 6.9},
		{Label: "low", Days: nil, CVSSMin: 0.0, CVSSMax: 3.9},
	}
}

func soc2PatchSLATiers() []SeverityTier {
	return []SeverityTier{
		{Label: "critical", Days: intP(30), CVSSMin: 9.0, CVSSMax: 10.0},
		{Label: "high", Days: intP(30), CVSSMin: 7.0, CVSSMax: 8.9},
		{Label: "moderate", Days: nil, CVSSMin: 4.0, CVSSMax: 6.9},
		{Label: "low", Days: nil, CVSSMin: 0.0, CVSSMax: 3.9},
	}
}

func intP(v int) *int {
	return &v
}

// idAliases maps seed-data / DB framework IDs (snake_case) to canonical registry keys.
// The seed data uses lowercase snake_case IDs (e.g. "cis", "pci_dss") while the registry
// uses the canonical display-style IDs (e.g. "CIS", "PCI-DSS"). Both forms are accepted.
var idAliases = map[string]string{
	"cis":         FrameworkCIS,
	"pci_dss":     FrameworkPCIDSSv4,
	"hipaa":       FrameworkHIPAA,
	"nist_800_53": FrameworkNIST80053,
	"iso_27001":   FrameworkISO27001,
	"soc_2":       FrameworkSOC2,
}

// GetFramework returns a shallow copy of the framework with the given ID, or nil if not found.
// It accepts both canonical IDs (e.g. "CIS") and seed-data aliases (e.g. "cis").
func GetFramework(id string) *Framework {
	canonical := id
	if alias, ok := idAliases[id]; ok {
		canonical = alias
	}
	fw, ok := registry[canonical]
	if !ok {
		return nil
	}
	cp := *fw
	return &cp
}

// ListFrameworks returns shallow copies of all registered frameworks.
func ListFrameworks() []*Framework {
	result := make([]*Framework, 0, len(registry))
	for _, fw := range registry {
		cp := *fw
		result = append(result, &cp)
	}
	return result
}
