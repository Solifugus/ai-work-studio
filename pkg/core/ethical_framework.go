package core

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/Solifugus/ai-work-studio/pkg/llm"
	"github.com/Solifugus/ai-work-studio/pkg/storage"
)

// DecisionUrgency represents how urgent an ethical decision is.
type DecisionUrgency int

const (
	// DecisionUrgencyLow for routine decisions with minimal ethical implications
	DecisionUrgencyLow DecisionUrgency = iota
	// DecisionUrgencyMedium for decisions with moderate ethical considerations
	DecisionUrgencyMedium
	// DecisionUrgencyHigh for decisions with significant ethical impact
	DecisionUrgencyHigh
	// DecisionUrgencyCritical for decisions that could severely impact user freedom or well-being
	DecisionUrgencyCritical
)

// DecisionOutcome represents the result of implementing an ethical decision.
type DecisionOutcome string

const (
	// DecisionOutcomeUnknown means we haven't tracked the outcome yet
	DecisionOutcomeUnknown DecisionOutcome = "unknown"
	// DecisionOutcomePositive means the decision had good results
	DecisionOutcomePositive DecisionOutcome = "positive"
	// DecisionOutcomeNeutral means the decision had minimal impact
	DecisionOutcomeNeutral DecisionOutcome = "neutral"
	// DecisionOutcomeNegative means the decision had harmful results
	DecisionOutcomeNegative DecisionOutcome = "negative"
)

// DecisionApprovalStatus represents whether a decision needs or has user approval.
type DecisionApprovalStatus string

const (
	// DecisionApprovalNotRequired for routine decisions that can proceed automatically
	DecisionApprovalNotRequired DecisionApprovalStatus = "not_required"
	// DecisionApprovalPending for decisions awaiting user approval
	DecisionApprovalPending DecisionApprovalStatus = "pending"
	// DecisionApprovalApproved for decisions the user has approved
	DecisionApprovalApproved DecisionApprovalStatus = "approved"
	// DecisionApprovalRejected for decisions the user has rejected
	DecisionApprovalRejected DecisionApprovalStatus = "rejected"
)

// EthicalImpact represents the assessed impact of a decision on ethical dimensions.
type EthicalImpact struct {
	// FreedomImpact scores impact on user's freedom (-1.0 to +1.0)
	// Negative values restrict freedom, positive values enhance it
	FreedomImpact float64

	// WellBeingImpact scores impact on user's well-being (-1.0 to +1.0)
	// Negative values harm well-being, positive values improve it
	WellBeingImpact float64

	// SustainabilityImpact scores impact on system sustainability (-1.0 to +1.0)
	// Negative values harm long-term viability, positive values improve it
	SustainabilityImpact float64

	// ConfidenceScore represents how confident we are in this assessment (0.0 to 1.0)
	ConfidenceScore float64

	// Reasoning explains the rationale behind these scores
	Reasoning string
}

// EthicalDecision represents a decision point that requires ethical evaluation.
// It tracks the decision, its ethical assessment, user feedback, and outcomes.
type EthicalDecision struct {
	// ID uniquely identifies this ethical decision
	ID string

	// ObjectiveID links this decision to the objective that triggered it
	ObjectiveID string

	// DecisionContext describes what decision needs to be made
	DecisionContext string

	// ProposedAction describes what the system wants to do
	ProposedAction string

	// AlternativeActions lists other possible actions considered
	AlternativeActions []string

	// Impact contains the ethical impact assessment
	Impact EthicalImpact

	// Urgency indicates how urgent this decision is
	Urgency DecisionUrgency

	// ApprovalStatus tracks whether user approval is needed/obtained
	ApprovalStatus DecisionApprovalStatus

	// UserFeedback contains any feedback the user provided about this decision
	UserFeedback string

	// Outcome tracks the actual result after implementing the decision
	Outcome DecisionOutcome

	// CreatedAt is when this decision was created
	CreatedAt time.Time

	// ApprovedAt is when the user approved this decision (if applicable)
	ApprovedAt *time.Time

	// ImplementedAt is when the decision was implemented
	ImplementedAt *time.Time

	// UserID identifies which user this decision belongs to
	UserID string

	// store reference for database operations
	store *storage.Store
}

// EthicalFramework provides ethical decision evaluation and tracking.
// It uses LLM-based reasoning to assess decisions against the Prime Value
// of Mutual Freedom and Well-Being.
type EthicalFramework struct {
	store           *storage.Store
	llmRouter       *llm.Router
	contextManager  *UserContextManager

	// Configuration for ethical reasoning
	freedomWeight      float64 // Weight given to freedom considerations (0-1)
	wellBeingWeight    float64 // Weight given to well-being considerations (0-1)
	sustainabilityWeight float64 // Weight given to sustainability considerations (0-1)
	approvalThreshold  float64 // Threshold below which user approval is required
}

// EthicalConfig contains configuration for the ethical framework.
type EthicalConfig struct {
	FreedomWeight        float64
	WellBeingWeight      float64
	SustainabilityWeight float64
	ApprovalThreshold    float64
}

// DefaultEthicalConfig returns sensible defaults for ethical framework configuration.
func DefaultEthicalConfig() EthicalConfig {
	return EthicalConfig{
		FreedomWeight:        0.4,  // Freedom is primary concern
		WellBeingWeight:      0.35, // Well-being is secondary
		SustainabilityWeight: 0.25, // Sustainability ensures long-term viability
		ApprovalThreshold:    0.6,  // Require approval if overall score < 0.6
	}
}

// NewEthicalFramework creates a new ethical decision framework.
func NewEthicalFramework(store *storage.Store, llmRouter *llm.Router, contextManager *UserContextManager, config ...EthicalConfig) *EthicalFramework {
	cfg := DefaultEthicalConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return &EthicalFramework{
		store:               store,
		llmRouter:           llmRouter,
		contextManager:      contextManager,
		freedomWeight:       cfg.FreedomWeight,
		wellBeingWeight:     cfg.WellBeingWeight,
		sustainabilityWeight: cfg.SustainabilityWeight,
		approvalThreshold:   cfg.ApprovalThreshold,
	}
}

// EvaluateDecision performs ethical evaluation of a proposed decision.
// It uses LLM-based reasoning to assess the decision against ethical principles.
func (ef *EthicalFramework) EvaluateDecision(ctx context.Context, objectiveID, decisionContext, proposedAction string, alternatives []string, userID string) (*EthicalDecision, error) {
	if decisionContext == "" {
		return nil, fmt.Errorf("decision context cannot be empty")
	}

	if proposedAction == "" {
		return nil, fmt.Errorf("proposed action cannot be empty")
	}

	// Get relevant user context for ethical reasoning
	userContext, err := ef.contextManager.GetRelevantContext(ctx, decisionContext+" "+proposedAction, userID, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to get user context: %w", err)
	}

	// Perform ethical reasoning using LLM
	impact, err := ef.performEthicalReasoning(ctx, decisionContext, proposedAction, alternatives, userContext)
	if err != nil {
		return nil, fmt.Errorf("failed to perform ethical reasoning: %w", err)
	}

	// Determine urgency based on impact scores
	urgency := ef.determineUrgency(impact)

	// Determine if approval is needed
	approvalStatus := ef.determineApprovalNeeded(impact, urgency)

	now := time.Now()

	// Create decision record
	decision := &EthicalDecision{
		ObjectiveID:        objectiveID,
		DecisionContext:    decisionContext,
		ProposedAction:     proposedAction,
		AlternativeActions: alternatives,
		Impact:             *impact,
		Urgency:            urgency,
		ApprovalStatus:     approvalStatus,
		Outcome:            DecisionOutcomeUnknown,
		CreatedAt:          now,
		UserID:             userID,
		store:              ef.store,
	}

	// Store the decision
	if err := ef.storeDecision(ctx, decision); err != nil {
		return nil, fmt.Errorf("failed to store decision: %w", err)
	}

	return decision, nil
}

// performEthicalReasoning uses LLM to assess ethical impact of a decision.
func (ef *EthicalFramework) performEthicalReasoning(ctx context.Context, decisionContext, proposedAction string, alternatives []string, userContext []*UserContext) (*EthicalImpact, error) {
	// Build context information from user context
	contextInfo := ef.buildContextInfo(userContext)

	// Create structured prompt for ethical reasoning
	prompt := ef.buildEthicalPrompt(decisionContext, proposedAction, alternatives, contextInfo)

	// Execute LLM reasoning
	request := llm.TaskRequest{
		Prompt:           prompt,
		MaxTokens:        800,
		Temperature:      0.3, // Lower temperature for consistent ethical reasoning
		TaskType:         "ethical_analysis",
		QualityRequired:  llm.QualityPremium, // Ethical decisions require highest quality
	}

	result, err := ef.llmRouter.Route(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("LLM routing failed: %w", err)
	}

	if result.ExecutionResult == nil {
		return nil, fmt.Errorf("no result from LLM execution")
	}

	// Parse the LLM response to extract ethical impact scores
	impact, err := ef.parseEthicalResponse(result.ExecutionResult.Text)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ethical response: %w", err)
	}

	return impact, nil
}

// buildContextInfo creates a summary of relevant user context for ethical reasoning.
func (ef *EthicalFramework) buildContextInfo(userContext []*UserContext) string {
	if len(userContext) == 0 {
		return "No specific user context available."
	}

	var contextParts []string

	// Group context by category
	valueContext := make([]string, 0)
	constraintContext := make([]string, 0)
	preferenceContext := make([]string, 0)

	for _, ctx := range userContext {
		switch ctx.Category {
		case ContextCategoryValues:
			valueContext = append(valueContext, ctx.Content)
		case ContextCategoryConstraints:
			constraintContext = append(constraintContext, ctx.Content)
		case ContextCategoryPreferences:
			preferenceContext = append(preferenceContext, ctx.Content)
		}
	}

	if len(valueContext) > 0 {
		contextParts = append(contextParts, "User values: "+strings.Join(valueContext, "; "))
	}
	if len(constraintContext) > 0 {
		contextParts = append(contextParts, "User constraints: "+strings.Join(constraintContext, "; "))
	}
	if len(preferenceContext) > 0 {
		contextParts = append(contextParts, "User preferences: "+strings.Join(preferenceContext, "; "))
	}

	return strings.Join(contextParts, "\n")
}

// buildEthicalPrompt creates a structured prompt for ethical reasoning.
func (ef *EthicalFramework) buildEthicalPrompt(decisionContext, proposedAction string, alternatives []string, contextInfo string) string {
	prompt := `You are an ethical reasoning system evaluating decisions based on the Prime Value of Mutual Freedom and Well-Being. Your task is to assess a proposed action's impact on:

1. USER FREEDOM: The user's autonomy, choice, and control over their environment
2. USER WELL-BEING: The user's health, happiness, productivity, and overall flourishing
3. SYSTEM SUSTAINABILITY: The long-term viability and health of the AI system

DECISION CONTEXT:
` + decisionContext + `

PROPOSED ACTION:
` + proposedAction

	if len(alternatives) > 0 {
		prompt += `

ALTERNATIVE ACTIONS CONSIDERED:
`
		for i, alt := range alternatives {
			prompt += fmt.Sprintf("%d. %s\n", i+1, alt)
		}
	}

	prompt += `

USER CONTEXT:
` + contextInfo + `

EVALUATION INSTRUCTIONS:
Analyze the proposed action and provide scores from -1.0 to +1.0 for each dimension:

- Freedom Impact (-1.0 to +1.0): How does this affect the user's autonomy and choice?
  * Negative: Restricts options, removes control, creates dependencies
  * Positive: Increases options, enhances control, promotes independence

- Well-Being Impact (-1.0 to +1.0): How does this affect the user's overall flourishing?
  * Negative: Causes stress, reduces productivity, harms health/happiness
  * Positive: Reduces stress, improves productivity, enhances health/happiness

- Sustainability Impact (-1.0 to +1.0): How does this affect system long-term viability?
  * Negative: Creates technical debt, unsustainable patterns, resource waste
  * Positive: Improves maintainability, efficient resource use, healthy patterns

- Confidence (0.0 to 1.0): How confident are you in this assessment?

CRITICAL PRINCIPLES:
- Always choose freedom over convenience when they conflict
- Prioritize user agency and informed choice
- Consider both immediate and long-term impacts
- Be especially cautious with actions that reduce user control

REQUIRED OUTPUT FORMAT:
Freedom Impact: [score from -1.0 to +1.0]
Well-Being Impact: [score from -1.0 to +1.0]
Sustainability Impact: [score from -1.0 to +1.0]
Confidence: [score from 0.0 to 1.0]
Reasoning: [2-3 sentence explanation of the assessment]

Please provide your ethical evaluation:`

	return prompt
}

// parseEthicalResponse extracts impact scores from LLM response.
func (ef *EthicalFramework) parseEthicalResponse(response string) (*EthicalImpact, error) {
	lines := strings.Split(response, "\n")

	impact := &EthicalImpact{}
	var err error

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Freedom Impact:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(line, "Freedom Impact:"))
			impact.FreedomImpact, err = ef.parseScore(scoreStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse freedom impact: %w", err)
			}
		} else if strings.HasPrefix(line, "Well-Being Impact:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(line, "Well-Being Impact:"))
			impact.WellBeingImpact, err = ef.parseScore(scoreStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse well-being impact: %w", err)
			}
		} else if strings.HasPrefix(line, "Sustainability Impact:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(line, "Sustainability Impact:"))
			impact.SustainabilityImpact, err = ef.parseScore(scoreStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse sustainability impact: %w", err)
			}
		} else if strings.HasPrefix(line, "Confidence:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(line, "Confidence:"))
			impact.ConfidenceScore, err = ef.parseScore(scoreStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse confidence: %w", err)
			}
		} else if strings.HasPrefix(line, "Reasoning:") {
			impact.Reasoning = strings.TrimSpace(strings.TrimPrefix(line, "Reasoning:"))
			// Continue reading additional reasoning lines
			for i := len(lines) - 1; i > 0; i-- {
				nextLine := strings.TrimSpace(lines[i])
				if nextLine != "" && !strings.Contains(nextLine, ":") {
					impact.Reasoning += " " + nextLine
				}
			}
		}
	}

	// Validate that we got all required scores
	if impact.Reasoning == "" {
		return nil, fmt.Errorf("missing reasoning in LLM response")
	}

	return impact, nil
}

// parseScore extracts a numeric score from a string, handling various formats.
func (ef *EthicalFramework) parseScore(scoreStr string) (float64, error) {
	// Remove common non-numeric prefixes/suffixes
	scoreStr = strings.TrimSpace(scoreStr)
	scoreStr = strings.TrimPrefix(scoreStr, "[")
	scoreStr = strings.TrimSuffix(scoreStr, "]")

	// Try to parse as float
	var score float64
	n, err := fmt.Sscanf(scoreStr, "%f", &score)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("invalid score format: %s", scoreStr)
	}

	return score, nil
}

// determineUrgency assesses how urgent a decision is based on its ethical impact.
func (ef *EthicalFramework) determineUrgency(impact *EthicalImpact) DecisionUrgency {
	// Calculate weighted overall impact
	overallImpact := (impact.FreedomImpact * ef.freedomWeight) +
		(impact.WellBeingImpact * ef.wellBeingWeight) +
		(impact.SustainabilityImpact * ef.sustainabilityWeight)

	// Consider magnitude of impact (absolute value)
	impactMagnitude := math.Abs(overallImpact)

	// Consider individual dimension magnitudes
	maxImpact := math.Max(math.Abs(impact.FreedomImpact),
		math.Max(math.Abs(impact.WellBeingImpact), math.Abs(impact.SustainabilityImpact)))

	// Critical if major negative impact on freedom or well-being
	if (impact.FreedomImpact < -0.7 || impact.WellBeingImpact < -0.7) && impact.ConfidenceScore > 0.7 {
		return DecisionUrgencyCritical
	}

	// High urgency for significant impacts
	if impactMagnitude > 0.5 || maxImpact > 0.6 {
		return DecisionUrgencyHigh
	}

	// Medium urgency for moderate impacts
	if impactMagnitude > 0.2 || maxImpact > 0.3 {
		return DecisionUrgencyMedium
	}

	return DecisionUrgencyLow
}

// determineApprovalNeeded decides if user approval is required for a decision.
func (ef *EthicalFramework) determineApprovalNeeded(impact *EthicalImpact, urgency DecisionUrgency) DecisionApprovalStatus {
	// Calculate weighted overall score
	overallScore := (impact.FreedomImpact * ef.freedomWeight) +
		(impact.WellBeingImpact * ef.wellBeingWeight) +
		(impact.SustainabilityImpact * ef.sustainabilityWeight)

	// Always require approval for critical decisions
	if urgency == DecisionUrgencyCritical {
		return DecisionApprovalPending
	}

	// Require approval if overall score is below threshold
	if overallScore < ef.approvalThreshold {
		return DecisionApprovalPending
	}

	// Require approval if any dimension has significant negative impact
	if impact.FreedomImpact < -0.3 || impact.WellBeingImpact < -0.3 {
		return DecisionApprovalPending
	}

	// Require approval if confidence is low on high-impact decisions
	if urgency >= DecisionUrgencyHigh && impact.ConfidenceScore < 0.6 {
		return DecisionApprovalPending
	}

	return DecisionApprovalNotRequired
}

// storeDecision persists an ethical decision to storage.
func (ef *EthicalFramework) storeDecision(ctx context.Context, decision *EthicalDecision) error {
	// Prepare data for storage node
	data := map[string]interface{}{
		"objective_id":         decision.ObjectiveID,
		"decision_context":     decision.DecisionContext,
		"proposed_action":      decision.ProposedAction,
		"alternative_actions":  decision.AlternativeActions,
		"freedom_impact":       decision.Impact.FreedomImpact,
		"wellbeing_impact":     decision.Impact.WellBeingImpact,
		"sustainability_impact": decision.Impact.SustainabilityImpact,
		"confidence_score":     decision.Impact.ConfidenceScore,
		"reasoning":            decision.Impact.Reasoning,
		"urgency":              decision.Urgency.String(),
		"approval_status":      string(decision.ApprovalStatus),
		"user_feedback":        decision.UserFeedback,
		"outcome":              string(decision.Outcome),
		"created_at":           decision.CreatedAt.Format(time.RFC3339),
		"user_id":              decision.UserID,
	}

	if decision.ApprovedAt != nil {
		data["approved_at"] = decision.ApprovedAt.Format(time.RFC3339)
	}
	if decision.ImplementedAt != nil {
		data["implemented_at"] = decision.ImplementedAt.Format(time.RFC3339)
	}

	// Create storage node
	node := storage.NewNode("ethical_decision", data)
	decision.ID = node.ID

	// Store the node
	if err := ef.store.AddNode(ctx, node); err != nil {
		return fmt.Errorf("failed to store ethical decision: %w", err)
	}

	return nil
}

// GetDecision retrieves an ethical decision by ID.
func (ef *EthicalFramework) GetDecision(ctx context.Context, decisionID string) (*EthicalDecision, error) {
	node, err := ef.store.GetNode(ctx, decisionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve decision %s: %w", decisionID, err)
	}

	if node.Type != "ethical_decision" {
		return nil, fmt.Errorf("node %s is not an ethical decision (type: %s)", decisionID, node.Type)
	}

	return ef.nodeToEthicalDecision(node)
}

// ApproveDecision marks a decision as approved by the user.
func (ef *EthicalFramework) ApproveDecision(ctx context.Context, decisionID, userFeedback string) error {
	decision, err := ef.GetDecision(ctx, decisionID)
	if err != nil {
		return fmt.Errorf("failed to get decision for approval: %w", err)
	}

	if decision.ApprovalStatus != DecisionApprovalPending {
		return fmt.Errorf("decision %s is not pending approval (current status: %s)", decisionID, decision.ApprovalStatus)
	}

	now := time.Now()
	decision.ApprovalStatus = DecisionApprovalApproved
	decision.ApprovedAt = &now
	decision.UserFeedback = userFeedback

	return ef.updateDecisionInStorage(ctx, decision)
}

// RejectDecision marks a decision as rejected by the user.
func (ef *EthicalFramework) RejectDecision(ctx context.Context, decisionID, userFeedback string) error {
	decision, err := ef.GetDecision(ctx, decisionID)
	if err != nil {
		return fmt.Errorf("failed to get decision for rejection: %w", err)
	}

	if decision.ApprovalStatus != DecisionApprovalPending {
		return fmt.Errorf("decision %s is not pending approval (current status: %s)", decisionID, decision.ApprovalStatus)
	}

	decision.ApprovalStatus = DecisionApprovalRejected
	decision.UserFeedback = userFeedback

	return ef.updateDecisionInStorage(ctx, decision)
}

// ImplementDecision marks a decision as implemented and tracks outcomes.
func (ef *EthicalFramework) ImplementDecision(ctx context.Context, decisionID string) error {
	decision, err := ef.GetDecision(ctx, decisionID)
	if err != nil {
		return fmt.Errorf("failed to get decision for implementation: %w", err)
	}

	if decision.ApprovalStatus == DecisionApprovalPending {
		return fmt.Errorf("cannot implement decision %s: still pending approval", decisionID)
	}

	if decision.ApprovalStatus == DecisionApprovalRejected {
		return fmt.Errorf("cannot implement decision %s: was rejected by user", decisionID)
	}

	now := time.Now()
	decision.ImplementedAt = &now

	return ef.updateDecisionInStorage(ctx, decision)
}

// RecordOutcome records the actual outcome of implementing a decision for learning.
func (ef *EthicalFramework) RecordOutcome(ctx context.Context, decisionID string, outcome DecisionOutcome, feedback string) error {
	decision, err := ef.GetDecision(ctx, decisionID)
	if err != nil {
		return fmt.Errorf("failed to get decision for outcome recording: %w", err)
	}

	decision.Outcome = outcome
	if feedback != "" {
		decision.UserFeedback = feedback
	}

	// Learn from the outcome - update user context based on feedback
	if err := ef.learnFromOutcome(ctx, decision); err != nil {
		// Log error but don't fail - outcome recording is still valuable
		fmt.Printf("Warning: failed to learn from outcome: %v\n", err)
	}

	return ef.updateDecisionInStorage(ctx, decision)
}

// learnFromOutcome updates user context based on decision outcomes.
func (ef *EthicalFramework) learnFromOutcome(ctx context.Context, decision *EthicalDecision) error {
	if decision.UserFeedback == "" || decision.Outcome == DecisionOutcomeUnknown {
		return nil // No learning data available
	}

	// Extract lessons from the outcome
	contextContent := fmt.Sprintf("Decision outcome: %s. User feedback: %s. Context: %s",
		decision.Outcome, decision.UserFeedback, decision.DecisionContext)

	// Determine context category based on the type of learning
	category := ContextCategoryValues
	if strings.Contains(strings.ToLower(decision.UserFeedback), "prefer") {
		category = ContextCategoryPreferences
	} else if strings.Contains(strings.ToLower(decision.UserFeedback), "cannot") ||
		strings.Contains(strings.ToLower(decision.UserFeedback), "don't") {
		category = ContextCategoryConstraints
	}

	// Learn new context from feedback
	_, err := ef.contextManager.LearnContext(ctx, category, contextContent,
		ContextSourceFeedback, []string{"ethical_decision", "outcome", "feedback"}, decision.UserID)

	return err
}

// updateDecisionInStorage updates an ethical decision in storage.
func (ef *EthicalFramework) updateDecisionInStorage(ctx context.Context, decision *EthicalDecision) error {
	data := map[string]interface{}{
		"objective_id":         decision.ObjectiveID,
		"decision_context":     decision.DecisionContext,
		"proposed_action":      decision.ProposedAction,
		"alternative_actions":  decision.AlternativeActions,
		"freedom_impact":       decision.Impact.FreedomImpact,
		"wellbeing_impact":     decision.Impact.WellBeingImpact,
		"sustainability_impact": decision.Impact.SustainabilityImpact,
		"confidence_score":     decision.Impact.ConfidenceScore,
		"reasoning":            decision.Impact.Reasoning,
		"urgency":              decision.Urgency.String(),
		"approval_status":      string(decision.ApprovalStatus),
		"user_feedback":        decision.UserFeedback,
		"outcome":              string(decision.Outcome),
		"created_at":           decision.CreatedAt.Format(time.RFC3339),
		"user_id":              decision.UserID,
	}

	if decision.ApprovedAt != nil {
		data["approved_at"] = decision.ApprovedAt.Format(time.RFC3339)
	}
	if decision.ImplementedAt != nil {
		data["implemented_at"] = decision.ImplementedAt.Format(time.RFC3339)
	}

	return ef.store.UpdateNode(ctx, decision.ID, data)
}

// nodeToEthicalDecision converts a storage node to an EthicalDecision object.
func (ef *EthicalFramework) nodeToEthicalDecision(node *storage.Node) (*EthicalDecision, error) {
	if node == nil {
		return nil, fmt.Errorf("node is nil")
	}

	// Extract basic fields
	objectiveID, _ := node.Data["objective_id"].(string)
	decisionContext, _ := node.Data["decision_context"].(string)
	proposedAction, _ := node.Data["proposed_action"].(string)
	userID, _ := node.Data["user_id"].(string)

	// Extract alternative actions
	var alternatives []string
	if altData, ok := node.Data["alternative_actions"].([]interface{}); ok {
		for _, alt := range altData {
			if altStr, ok := alt.(string); ok {
				alternatives = append(alternatives, altStr)
			}
		}
	}

	// Extract impact scores
	impact := EthicalImpact{
		FreedomImpact:        getFloat64(node.Data, "freedom_impact"),
		WellBeingImpact:      getFloat64(node.Data, "wellbeing_impact"),
		SustainabilityImpact: getFloat64(node.Data, "sustainability_impact"),
		ConfidenceScore:      getFloat64(node.Data, "confidence_score"),
		Reasoning:            getString(node.Data, "reasoning"),
	}

	// Extract urgency
	urgencyStr := getString(node.Data, "urgency")
	urgency := parseUrgency(urgencyStr)

	// Extract approval status
	approvalStr := getString(node.Data, "approval_status")
	approvalStatus := DecisionApprovalStatus(approvalStr)

	// Extract outcome
	outcomeStr := getString(node.Data, "outcome")
	outcome := DecisionOutcome(outcomeStr)

	// Extract timestamps
	createdAtStr := getString(node.Data, "created_at")
	createdAt, _ := time.Parse(time.RFC3339, createdAtStr)

	var approvedAt *time.Time
	if approvedAtStr := getString(node.Data, "approved_at"); approvedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, approvedAtStr); err == nil {
			approvedAt = &t
		}
	}

	var implementedAt *time.Time
	if implementedAtStr := getString(node.Data, "implemented_at"); implementedAtStr != "" {
		if t, err := time.Parse(time.RFC3339, implementedAtStr); err == nil {
			implementedAt = &t
		}
	}

	userFeedback := getString(node.Data, "user_feedback")

	return &EthicalDecision{
		ID:                 node.ID,
		ObjectiveID:        objectiveID,
		DecisionContext:    decisionContext,
		ProposedAction:     proposedAction,
		AlternativeActions: alternatives,
		Impact:             impact,
		Urgency:            urgency,
		ApprovalStatus:     approvalStatus,
		UserFeedback:       userFeedback,
		Outcome:            outcome,
		CreatedAt:          createdAt,
		ApprovedAt:         approvedAt,
		ImplementedAt:      implementedAt,
		UserID:             userID,
		store:              ef.store,
	}, nil
}

// Helper functions for data extraction

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return ""
}

func getFloat64(data map[string]interface{}, key string) float64 {
	if val, ok := data[key].(float64); ok {
		return val
	}
	return 0.0
}

func parseUrgency(urgencyStr string) DecisionUrgency {
	switch urgencyStr {
	case "low":
		return DecisionUrgencyLow
	case "medium":
		return DecisionUrgencyMedium
	case "high":
		return DecisionUrgencyHigh
	case "critical":
		return DecisionUrgencyCritical
	default:
		return DecisionUrgencyLow
	}
}

// String methods for enums

func (du DecisionUrgency) String() string {
	switch du {
	case DecisionUrgencyLow:
		return "low"
	case DecisionUrgencyMedium:
		return "medium"
	case DecisionUrgencyHigh:
		return "high"
	case DecisionUrgencyCritical:
		return "critical"
	default:
		return "unknown"
	}
}

func (do DecisionOutcome) String() string {
	return string(do)
}

func (das DecisionApprovalStatus) String() string {
	return string(das)
}

// IsApproved returns true if the decision has been approved by the user.
func (ed *EthicalDecision) IsApproved() bool {
	return ed.ApprovalStatus == DecisionApprovalApproved
}

// IsRejected returns true if the decision has been rejected by the user.
func (ed *EthicalDecision) IsRejected() bool {
	return ed.ApprovalStatus == DecisionApprovalRejected
}

// IsPendingApproval returns true if the decision is waiting for user approval.
func (ed *EthicalDecision) IsPendingApproval() bool {
	return ed.ApprovalStatus == DecisionApprovalPending
}

// IsImplemented returns true if the decision has been implemented.
func (ed *EthicalDecision) IsImplemented() bool {
	return ed.ImplementedAt != nil
}

// GetOverallScore calculates the weighted overall ethical score for the decision.
func (ed *EthicalDecision) GetOverallScore(framework *EthicalFramework) float64 {
	return (ed.Impact.FreedomImpact * framework.freedomWeight) +
		(ed.Impact.WellBeingImpact * framework.wellBeingWeight) +
		(ed.Impact.SustainabilityImpact * framework.sustainabilityWeight)
}