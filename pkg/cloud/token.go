package cloud

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// k6-operator has to handle authentication tokens for Cloud tests.
// This struct contains key utilities to encapsulate that logic.
type TokenInfo struct {
	value string

	secretName      string
	secretNamespace string

	// Ready shows whether token was loaded yet or there should be a retry.
	// If it's false, either there was no attempt to load the token or there was
	// an attempt that ended in a recoverable error, and a caller should try again.
	Ready bool
}

const (
	// For all modes, k6-operator expects to find the token under this key.
	tokenSecretKey = "token"

	// In cloud output mode, k6-operator expects to find the token under
	// this label pair
	tokenLabelKey   = "k6cloud"
	tokenLabelValue = "token"
)

func NewTokenInfo(name, namespace string) *TokenInfo {
	return &TokenInfo{
		secretName:      name,
		secretNamespace: namespace,
	}
}

func (ti TokenInfo) SecretName() string {
	return ti.secretName
}

func (ti TokenInfo) Value() string {
	return ti.value
}

// Load attempts to load the Secret and populate TokenInfo.
// returnErr means a non-recoverable error and requires the caller to take action or
// propagate it further.
func (ti *TokenInfo) Load(ctx context.Context, log logr.Logger, c k8sClient.Client) (returnErr error) {
	var (
		secrets corev1.SecretList
		secret  corev1.Secret
	)

	if len(ti.secretName) == 0 {
		// A bit of a hack: if we don't know the name of secret, it's
		// likely a cloud output mode, which expects to load the token from
		// k6-operator-system namespace with a label pair.
		ti.secretNamespace = "k6-operator-system"
	}

	if len(ti.secretName) > 0 {
		log.Info("Loading token by name.", "name", ti.secretName, "secretNamespace", ti.secretNamespace)

		if err := c.Get(ctx, types.NamespacedName{Namespace: ti.secretNamespace, Name: ti.secretName}, &secret); err != nil {
			log.Error(err, "Failed to load k6 Cloud token", "name", ti.secretName, "secretNamespace", ti.secretNamespace)
			// This may be a networking issue, etc. so just retry.
			return
		}
	} else {
		secretOpts := &k8sClient.ListOptions{
			Namespace: ti.secretNamespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				tokenLabelKey: tokenLabelValue,
			}),
		}

		log.Info("Loading token by label pair.", "labelset", secretOpts.LabelSelector.String(), "secretNamespace", secretOpts.Namespace)

		if err := c.List(ctx, &secrets, secretOpts); err != nil {
			log.Error(err, "Failed to load k6 Cloud token", "labelset", secretOpts.LabelSelector.String(), "secretNamespace", secretOpts.Namespace)
			// This may be a networking issue, etc. so just retry.
			return
		}

		if len(secrets.Items) < 1 {
			// we should stop execution in case of this error
			returnErr = fmt.Errorf("no secret with k6 Cloud token found")
			log.Error(returnErr, returnErr.Error(), "labelset", secretOpts.LabelSelector.String(), "secretNamespace", secretOpts.Namespace)
			return
		}

		secret = secrets.Items[0]
	}

	if t, ok := secret.Data[tokenSecretKey]; !ok {
		// we should stop execution in case of this error
		returnErr = fmt.Errorf("the secret doesn't have a field `%s` for k6 Cloud token", tokenSecretKey)
		log.Error(returnErr, returnErr.Error())
		return
	} else {
		ti.value = string(t)
		ti.Ready = true
		log.Info("Token for k6 Cloud was loaded.")
	}

	return
}

// InjectValue: this is only for unit tests!
// If you see it elsewhere, it's likely a bug or an attack.
func (ti *TokenInfo) InjectValue(v string) *TokenInfo {
	ti.value = v
	return ti
}
