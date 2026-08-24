// Tier 7 Phase 2 — Security (D6) — Compliance dashboard.
//
//	GET /api/v1/security/compliance — per-framework pass/fail tally + rule list
//
// Returns one summary per framework (PCI / SOC2 / GDPR / HIPAA) with:
//   - total_rules: count of rules seeded/defined for that framework
//   - pass_count / fail_count / unknown_count: rollup of the most-recent
//     compliance_results row per rule
//   - rules: per-rule detail with the most-recent evaluation status
//
// Phase 2 ships a read-only view. Rule mutation + evaluation are future
// changes. HIPAA rules can exist but aren't seeded (PCI/SOC2/GDPR are
// the only ones the seed migration inserts).
package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/stackwatch/platform/internal/db"
	"github.com/stackwatch/platform/internal/kernel"
)

// complianceRule is one row in the rules[] array on each framework.
type complianceRule struct {
	RuleID      string `json:"rule_id"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
	Remediation string `json:"remediation,omitempty"`
	Status      string `json:"status"` // 'pass' | 'fail' | 'unknown'
	Evidence    string `json:"evidence,omitempty"`
}

// complianceFramework is one entry in the dashboard's frameworks[] array.
type complianceFramework struct {
	Framework    string           `json:"framework"`
	TotalRules   int              `json:"total_rules"`
	PassCount    int              `json:"pass_count"`
	FailCount    int              `json:"fail_count"`
	UnknownCount int              `json:"unknown_count"`
	Rules        []complianceRule `json:"rules"`
}

// ListSecurityCompliance returns the compliance dashboard for the
// caller's tenant. We use a single CTE that picks the most-recent
// compliance_results row per (rule_id, resource_id) pair so the
// dashboard never double-counts old evaluations.
func ListSecurityCompliance(pool *db.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID, ok := tenantIDFromContext(c)
		if !ok {
			kernel.RespondError(c, kernel.ErrUnauthorized)
			return
		}
		// Pull every rule for this tenant + the most-recent result.
		// The rules LEFT JOIN on the latest evaluation so rules with
		// no evaluation show up as 'unknown' (correct behavior for
		// newly-seeded frameworks).
		rows, err := pool.Pgx().Query(c.Request.Context(),
			`WITH latest AS (
			   SELECT DISTINCT ON (rule_id)
			          rule_id, status, COALESCE(evidence, '') AS evidence
			     FROM compliance_results
			    WHERE tenant_id = $1
			    ORDER BY rule_id, evaluated_at DESC
			 )
			 SELECT r.framework, r.rule_id, r.description, r.severity,
			        COALESCE(r.remediation, ''),
			        COALESCE(l.status, 'unknown'),
			        COALESCE(l.evidence, '')
			   FROM compliance_rules r
			   LEFT JOIN latest l ON l.rule_id = r.id
			  WHERE r.tenant_id = $1
			  ORDER BY r.framework ASC, r.rule_id ASC`,
			tenantID,
		)
		if err != nil {
			kernel.RespondError(c, err)
			return
		}
		defer rows.Close()

		// Bucket per-framework in Go so we don't have to do it in SQL
		// (4 frameworks × ≤20 rules each — Go aggregation is trivial
		// and keeps the SQL readable).
		type bucket struct {
			fw    complianceFramework
			rules []complianceRule
		}
		seen := map[string]*bucket{}
		order := []string{}
		for rows.Next() {
			var framework, ruleID, description, severity, remediation, status, evidence string
			if err := rows.Scan(&framework, &ruleID, &description, &severity,
				&remediation, &status, &evidence); err != nil {
				continue
			}
			b, ok := seen[framework]
			if !ok {
				b = &bucket{
					fw: complianceFramework{
						Framework: framework,
						Rules:     []complianceRule{},
					},
				}
				seen[framework] = b
				order = append(order, framework)
			}
			b.fw.TotalRules++
			switch status {
			case "pass":
				b.fw.PassCount++
			case "fail":
				b.fw.FailCount++
			default:
				b.fw.UnknownCount++
			}
			b.fw.Rules = append(b.fw.Rules, complianceRule{
				RuleID:      ruleID,
				Description: description,
				Severity:    severity,
				Remediation: remediation,
				Status:      status,
				Evidence:    evidence,
			})
		}

		out := make([]complianceFramework, 0, len(order))
		for _, fw := range order {
			out = append(out, seen[fw].fw)
		}
		kernel.RespondOK(c, gin.H{"frameworks": out, "total": len(out)})
	}
}
