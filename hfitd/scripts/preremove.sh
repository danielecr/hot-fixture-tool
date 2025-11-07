#!/bin/bash
# Stop and disable service
systemctl stop hfitd.service || true
systemctl disable hfitd.service || true