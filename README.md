# jaal

A modular GraphQL framework in Go.

## Useful Tips

### Subscriptions

    - Set return value to pointer type for each subscription type resolver and return nil if you don't want the client to recieve null responses