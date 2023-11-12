# auth0-operator

Kubernetes operator for Auth0

## Example Client

```yaml
apiVersion: auth0.gracey.io/v1alpha1
kind: Client
metadata:
    name: client-sample
spec:
    name: auth0-operator-sample
    description: Auth0 Operator Sample client
    type: spa
    callbackUrls:
        - http://localhost:3000/callback
        - https://example.com/callback
    metadata:
        something: placeholder value
    clientSecret:
        # Must be at least 48 characters long
        literal: "some-secretsome-secretsome-secretsome-secrjshdnd"

        # SecretRef takes precedence over literal
        secretRef:
            name: client-secret
            key: something

        # Optional. Output the client secret to a kuberenetes secret
        outputSecretRef:
            name: output-client-secret
            key: output-client-secret
```
