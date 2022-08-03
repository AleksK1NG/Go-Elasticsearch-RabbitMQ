package v1

func (h *productController) MapRoutes() {
	h.group.POST("", h.index())
}
