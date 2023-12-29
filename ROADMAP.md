# Roadmap

## Stuff to add

### Code

- Support passing named Envs through flags (`-e foo=bar`) to command templates (`{{ .Env.foo }}` evaluates to `bar`)
- Add a switch allowing to grep file path or file contents
- Add a switch allowing to select only directories or only files
- (Cosmetic) Implement an action capable of initializing a lightweight `baraddur.yml` template file.
- (Cosmetic) Implement a counter that allows to track completion
   **Note**: This probably needs to be implemented as a separate channel and some counters

### Tests

- Test the code

### Delivery

- GitHub actions
- Publishing of releases (Need to understand `goreleaser` first...)
- Publishing of Docker Image

### Other

- A proper README
