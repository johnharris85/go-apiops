plugins:
  - name: ai-mcp-proxy
    route: airmiles-mcp-route
    config:
      logging:
        log_statistics: true
      mode: conversion-listener
      tools:
      - description: Get tiers
        method: GET
        path: "/miles/tiers"
      - description: Get member by id
        method: GET
        path: "/miles/members/{id}"
        parameters:
        - name: id
          in: path
          required: false
          schema:
            type: string
          description: Optional member ID
      server:
        timeout: 60000
