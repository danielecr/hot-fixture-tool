#!/bin/bash
# Create hfitd user
if ! id hfitd >/dev/null 2>&1; then
  useradd --system --home /var/lib/hfitd --shell /sbin/nologin hfitd
fi