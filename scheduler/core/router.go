package core

import (
	"github.com/gin-gonic/gin"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
)

func newControlRouter(state *internalState, fidStore store.Singleton) *gin.Engine {
	controlRouter := gin.Default()
	controlRouter.GET("/status", func(c *gin.Context){
		state.RLock()
		defer state.RUnlock()
		msg := gin.H{
			"tasksLaunched": state.tasksLaunched,
			"tasksFinished": state.tasksFinished,
			"totalTasks": state.totalTasks,
			"frameworkId": store.GetIgnoreErrors(fidStore)(),
			"config" : state.config,
		}
		if state.config.verbose {
			c.IndentedJSON(200, msg)
		} else {
			c.JSON(200, msg)
		}
	})
	return controlRouter
}