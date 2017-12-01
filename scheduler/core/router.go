package core

import (
	"github.com/gin-gonic/gin"
	"github.com/mesos/mesos-go/api/v1/lib/extras/store"
	"gitlab.cern.ch/tmrnjava/test-scheduler/scheduler/core/environment"
	"net/http"
	"fmt"
	"github.com/gin-gonic/gin/json"
	"time"
	"github.com/sirupsen/logrus"
	"gitlab.cern.ch/tmrnjava/test-scheduler/scheduler/logger"
)

// ginLogHandler returns a handler function for gin, which logs all events to the
// main logger with the correct prefix and fields.
// Based on ginrus from gin-gonic/contrib.
func ginLogHandler(logger *logger.Log, timeFormat string, utc bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		// some evil middlewares modify this values
		path := c.Request.URL.Path
		c.Next()

		end := time.Now()
		latency := end.Sub(start)
		if utc {
			end = end.UTC()
		}

		entry := logger.WithPrefix("http-router").WithFields(logrus.Fields{
			"status":     c.Writer.Status(),
			"method":     c.Request.Method,
			"path":       path,
			"ip":         c.ClientIP(),
			"latency":    latency,
			"user-agent": c.Request.UserAgent(),
			"time":       end.Format(timeFormat),
		})

		if len(c.Errors) > 0 {
			// Append error field if this is an erroneous request.
			entry := entry.WithField("error", c.Errors.String())
			entry.Error("gin error")
		} else {
			entry.Info("gin request")
		}
	}
}

// niy returns a generic not-implemented-yet response.
func niy(c *gin.Context) {
	msg := gin.H{
		"message": "Not implemented yet :(",
		"method":  c.Request.Method,
		"path":    c.Request.RequestURI,
	}
	c.IndentedJSON(http.StatusOK, msg)
}

// errorResponse returns a 500 internal server error response, with the error message.
func errorResponse(msg string, c *gin.Context) {
	log.WithPrefix("http-router").WithField("error", msg).Warn("preparing error response")
	c.JSON(http.StatusInternalServerError, gin.H{"error": msg})
	return
}

// instantiates a new Gin HTTP router for the control API.
func newControlRouter(state *internalState, fidStore store.Singleton) *gin.Engine {
	controlRouter := gin.New()
	controlRouter.Use(ginLogHandler(log, time.RFC3339, false), gin.Recovery())

	controlRouter.GET("/status", get_status(state, fidStore))
	controlRouter.DELETE("/status", delete_status(state, fidStore))
	controlRouter.GET("/environments", get_environments(state, fidStore))
	controlRouter.POST("/environments", post_environments(state, fidStore))
	controlRouter.GET("/environments/:id", get_environments_id(state, fidStore))
	controlRouter.POST("/environments/:id", post_environments_id(state, fidStore))
	return controlRouter
}

func get_status(state *internalState, fidStore store.Singleton) gin.HandlerFunc {
	return func(c *gin.Context) {
		state.RLock()
		defer state.RUnlock()
		msg := gin.H{
			"tasksLaunched": state.tasksLaunched,
			"tasksFinished": state.tasksFinished,
			"frameworkId":   store.GetIgnoreErrors(fidStore)(),
			"config":        state.config,
			"currentState":  state.sm.Current(),
		}
		if state.config.verbose {
			c.IndentedJSON(http.StatusOK, msg)
		} else {
			c.JSON(http.StatusOK, msg)
		}
	}
}

func delete_status(state *internalState, fidStore store.Singleton) gin.HandlerFunc {
	return func(c *gin.Context) {
		//TODO: implement teardown

	}
}

func get_environments(state *internalState, fidStore store.Singleton) gin.HandlerFunc {
	return func(c *gin.Context) {
		response := gin.H{
			"frameworkId": store.GetIgnoreErrors(fidStore)(),
			"environments": func() (envsList []gin.H) {
				ids := state.environments.Ids()
				envsList = make([]gin.H, len(ids))
				for i, id := range ids {
					envsList[i] = gin.H{
						"id": id,
						"configuration": state.environments.Configuration(id),
					}
				}
				return
			}(),
		}
		c.JSON(http.StatusOK, response)
	}
}

func post_environments(state *internalState, fidStore store.Singleton) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cfg environment.Configuration
		if err := c.ShouldBindJSON(&cfg); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			log.WithPrefix("http-router").WithField("error", err.Error()).Error("cannot parse JSON request")
			return
		}

		if state.config.verbose {
			payload, _ := json.Marshal(cfg)
			log.WithPrefix("http-router").WithField("payload", payload).Debug("received JSON payload")
		}

		// NEW_ENVIRONMENT transition
		// The following should
		// 1) Create a new value of type Environment struct
		// 2) Build the topology and ask Mesos to run all the processes
		// 3) Acquire the status of the processes to ascertain that they are indeed running and
		//    in their STANDBY state
		// 4) Execute the CONFIGURE transition on all the processes, and recheck their status to
		//    make sure they are now successfully in CONFIGURED
		// 5) Report back here with the new environment id and error code, if needed.

		id, err := state.environments.CreateNew(cfg)
		if err != nil {
			errorResponse(err.Error(), c)
			return
		}

		if state.sm.Cannot("NEW_ENVIRONMENT") {
			msg := fmt.Sprintf("NEW_ENVIRONMENT transition impossible, current state: %s",
				state.sm.Current())
			errorResponse(msg, c)
			return
		}

		state.sm.Event("NEW_ENVIRONMENT") //Async until Transition call

		newEnv, err := state.environments.Environment(id)
		if err != nil {
			errorResponse(err.Error(), c)
			return
		}
		newEnv.Sm.Event("CONFIGURE") //Async until Transition call


		// 1 is done, next up: 2) build topology and ask Mesos to run.
		// In order to do this, we need to kludge something simple, but first learn about Mesos
		// labels, roles and reservations so we don't do something stupid.


		//idea: a flps mesos-role assigned to all mesos agents on flp hosts, and then a static
		//      reservation for that mesos-role on behalf of our scheduler









		// First ask scheduler whether stuff is running and ok, then
		newEnv.Sm.Transition()


		msg := gin.H{
			"message":       "",
			"error":         0, //TODO make this meaningful
			"environmentId": id,
			"frameworkId":   store.GetIgnoreErrors(fidStore)(),
		}
		if state.config.verbose {
			c.IndentedJSON(http.StatusOK, msg)
		} else {
			c.JSON(http.StatusOK, msg)
		}

		// This POST should acquire a JSON payload with the topology to deploy, and then we should
		// c.ShouldBindJSON

		// Finally we let the main FSM complete the NEW_ENVIRONMENT transition
		state.sm.Transition()
	}
}

func get_environments_id(state *internalState, fidStore store.Singleton) gin.HandlerFunc {
	return func(c *gin.Context) {
		//environmentId := c.Param("id")

		niy(c)
	}
}

func post_environments_id(state *internalState, fidStore store.Singleton) gin.HandlerFunc {
	return func(c *gin.Context) {
		// This is the main control entry point for an existing environment.
		// A parameter (TBD) will have the name (and maybe arguments) for triggering an event in the
		// environment's FSM. Then we do something like:
		//	if state.environment[id].sm.Can(eventName) {
		//		state.environment[id].sm.Event(eventName)
		//
		// For the POST payload, we should bind it to a struct which encapsulates the parameters
		// for the relevant transition events.

		niy(c)
	}
}