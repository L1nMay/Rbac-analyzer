package httpapi

import (
	"io"
	"net/http"
)

// Это заготовка под Stripe webhook.
// Для продаваемого проекта это обязательно, но подключение Stripe требует секретов и настройки.
// В MVP оставляем endpoint, который логирует payload (позже добавим сигнатуры Stripe).
func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	_ = body // тут ты потом проверишь Stripe-Signature и обновишь subscriptions

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
