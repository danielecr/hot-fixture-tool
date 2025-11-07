#!/bin/bash
# Create directories and set permissions
systemd-tmpfiles --create hfitd.conf
# Reload systemd
systemctl daemon-reload
# Enable but don't start the service
systemctl enable hfitd.service
echo "Configuration file: /etc/hfitd/.env"
echo "Start service: systemctl start hfitd.service"