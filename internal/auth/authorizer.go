package auth

import (
	"fmt"
	"github.com/casbin/casbin/v2"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"log"
)

type Authorizer struct {
	enforcer *casbin.Enforcer
}

func New(model, policy string) *Authorizer {
	enforcer, err := casbin.NewEnforcer(model, policy)
	if err != nil {
		log.Fatalf("err: enforcer: %s", err)
	}

	return &Authorizer{
		enforcer: enforcer,
	}
}

func (a *Authorizer) Authorize(subject, object, action string) error {
	if ok, err := a.enforcer.Enforce(subject, object, action); err != nil {
		return err
	} else if !ok {
		msg := fmt.Sprintf(
			"%s not permitted to %s to %s",
			subject,
			action,
			object,
		)
		st := status.New(codes.PermissionDenied, msg)
		return st.Err()
	}
	return nil
}

//func (a *Authorizer) Authorize(subject, action, object string) error {
//	allow, err := a.enforcer.Enforce(subject, object, action)
//	if err != nil {
//		return err
//	}
//	if !allow {
//		msg := fmt.Sprintf("%s not permitted to %s to %s", subject, action, object)
//		st := status.New(codes.PermissionDenied, msg)
//		return st.Err()
//	}
//	return nil
//}
