```yaml
apiVersion: auth0.gracey.io/v1alpha1
kind: Client
metadata:
    name: client-sample
spec:
    # Required. The name of Auth0 client
    name: auth0-operator-sample
    # Optional. The description of Auth0 client
    description: Auth0 Operator Sample client
    # Required. The type of Auth0 client
    type: spa

    # Optional. The URLs that Auth0 is allowed to redirect to after authentication
    callbackUrls:
        - http://localhost:3000/callback
        - https://example.com/callback

    # Optional. Metadata to be included in the client
    metadata:
        something: placeholder value

    # Optional. Supply the client secret as either a literal value or as
    # a kubernetes secret. secretRef takes precedence over literal.
    # If neither are supplied, a secret will be generated and output to
    # outputSecretRef if supplied.
    clientSecret:
        # Optional. Supply the client secret as a literal value
        # Must be at least 48 characters long
        literal: "some-secretsome-secretsome-secretsome-secrjshdnd"

        # Optional. Supply the client secret as a kubernetes secret
        secretRef:
            name: client-secret
            key: something

        # Optional. Output the client secret to a kuberenetes secret
        outputSecretRef:
            name: output-client-secret
            key: output-client-secret
```
