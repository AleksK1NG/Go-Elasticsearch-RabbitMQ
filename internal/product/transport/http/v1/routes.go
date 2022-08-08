package v1

func (h *productController) MapRoutes() {
	h.group.POST("", h.index())
	h.group.POST("/async", h.indexAsync())
	h.group.GET("/search", h.search())
}
