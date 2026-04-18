package nexusggr

type ProviderListResponse struct {
	Status    int              `json:"status"`
	Msg       string           `json:"msg,omitempty"`
	Providers []map[string]any `json:"providers,omitempty"`
}

type GameListResponse struct {
	Status int              `json:"status"`
	Msg    string           `json:"msg,omitempty"`
	Games  []map[string]any `json:"games,omitempty"`
}

type GameLaunchResponse struct {
	Status    int    `json:"status"`
	Msg       string `json:"msg,omitempty"`
	LaunchURL any    `json:"launch_url,omitempty"`
}

type MoneyInfoResponse struct {
	Status   int              `json:"status"`
	Msg      string           `json:"msg,omitempty"`
	Agent    map[string]any   `json:"agent,omitempty"`
	User     map[string]any   `json:"user,omitempty"`
	UserList []map[string]any `json:"user_list,omitempty"`
}

type GameLogResponse struct {
	Status     int              `json:"status"`
	Msg        string           `json:"msg,omitempty"`
	TotalCount any              `json:"total_count,omitempty"`
	Page       any              `json:"page,omitempty"`
	PerPage    any              `json:"perPage,omitempty"`
	Slot       []map[string]any `json:"slot,omitempty"`
}

type UserCreateResponse struct {
	Status      int    `json:"status"`
	Msg         string `json:"msg,omitempty"`
	UserCode    any    `json:"user_code,omitempty"`
	UserBalance any    `json:"user_balance,omitempty"`
}

type UserBalanceMutationResponse struct {
	Status       int    `json:"status"`
	Msg          string `json:"msg,omitempty"`
	AgentBalance any    `json:"agent_balance,omitempty"`
	UserBalance  any    `json:"user_balance,omitempty"`
}

type UserWithdrawResetResponse struct {
	Status   int              `json:"status"`
	Msg      string           `json:"msg,omitempty"`
	Agent    map[string]any   `json:"agent,omitempty"`
	User     map[string]any   `json:"user,omitempty"`
	UserList []map[string]any `json:"user_list,omitempty"`
}

type TransferStatusResponse struct {
	Status       int    `json:"status"`
	Msg          string `json:"msg,omitempty"`
	Amount       any    `json:"amount,omitempty"`
	Type         any    `json:"type,omitempty"`
	AgentBalance any    `json:"agent_balance,omitempty"`
	UserBalance  any    `json:"user_balance,omitempty"`
}

type CallPlayersResponse struct {
	Status int              `json:"status"`
	Msg    string           `json:"msg,omitempty"`
	Data   []map[string]any `json:"data,omitempty"`
}

type CallListResponse struct {
	Status int              `json:"status"`
	Msg    string           `json:"msg,omitempty"`
	Calls  []map[string]any `json:"calls,omitempty"`
}

type CallApplyResponse struct {
	Status      int    `json:"status"`
	Msg         string `json:"msg,omitempty"`
	CalledMoney any    `json:"called_money,omitempty"`
}

type CallHistoryResponse struct {
	Status int              `json:"status"`
	Msg    string           `json:"msg,omitempty"`
	Data   []map[string]any `json:"data,omitempty"`
}

type CallCancelResponse struct {
	Status        int    `json:"status"`
	Msg           string `json:"msg,omitempty"`
	CanceledMoney any    `json:"canceled_money,omitempty"`
}

type ControlRtpResponse struct {
	Status     int    `json:"status"`
	Msg        string `json:"msg,omitempty"`
	ChangedRTP any    `json:"changed_rtp,omitempty"`
}
