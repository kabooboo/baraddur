# Roadmap

## New features

### Must-have

0. Execute some kind of script in bash in worker
1. Load root directory from command arguments
2. Load regexp and script to execute from config file
3. Implement an action capable of initializing a lightweight `baraddur.yml` file.
4. Implement a counter that allows to track completion
   **Note**: this needs to be implemented as a separate channel and some counters
5. Make it possible to ignore failure (or even raise critical if a given scan fails)
6. On workers with debug, print stderr, out as it goes

### Nice-to-have

1. CI mode versus "pretty print CLI" mode (color, completion bars, ...)
1.

## Tests

1. Everything

## Delivery

1. GitHub actions
2. Dockerfile
3. Publishing of releases (Understand `goreleaser`)
4. Publishing of Docker Image

## Docs

1. A proper README.
