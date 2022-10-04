package v1alpha1

import (
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var networkcounterlog = logf.Log.WithName("networkcounter-resource")

func (in *NetworkCounter) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(in).
		Complete()
}

// +kubebuilder:webhook:path=/validate-ipam-onmetal-de-v1alpha1-networkcounter,mutating=false,failurePolicy=fail,sideEffects=None,groups=ipam.onmetal.de,resources=networkcounters,verbs=create;update;delete,versions=v1alpha1,name=vnetworkcounter.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &NetworkCounter{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (in *NetworkCounter) ValidateCreate() error {
	networkcounterlog.Info("validate create", "name", in.Name)
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (in *NetworkCounter) ValidateUpdate(_ runtime.Object) error {
	networkcounterlog.Info("validate update", "name", in.Name)
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (in *NetworkCounter) ValidateDelete() error {
	networkcounterlog.Info("validate delete", "name", in.Name)

	var allErrs field.ErrorList

	if len(in.Spec.Vacant) == 0 {
		allErrs = append(allErrs, field.InternalError(field.NewPath("metadata.name"), errors.New("Network Counter is still in use by networks")))
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "NetworkCounter",
			}, in.Name, allErrs)
	}

	begin := in.Spec.Vacant[0].Begin
	end := in.Spec.Vacant[0].End

	if end == nil && begin.Eq(CMPLSFirstAvailableID) {
		return nil
	}

	if begin.Eq(CVXLANFirstAvaliableID) && end.Eq(CVXLANMaxID) {
		return nil
	}

	allErrs = append(allErrs, field.InternalError(field.NewPath("metadata.name"), errors.New("Network Counter is still in use by networks")))
	return apierrors.NewInvalid(
		schema.GroupKind{
			Group: GroupVersion.Group,
			Kind:  "NetworkCounter",
		}, in.Name, allErrs)
}
