# PlexiChat API Design - Production Ready Endpoints

## Overview
This document outlines the improved REST API design for PlexiChat with proper RESTful structure, versioning, security, and scalability considerations.

## Base URL Structure
```
https://api.plexichat.com/v1/
```

## Authentication
- **JWT Bearer Tokens**: Primary authentication method
- **API Keys**: For service-to-service communication
- **2FA/MFA Support**: TOTP, SMS, Email, Hardware keys

## Core Endpoint Categories

### 1. Authentication & Authorization
```
POST   /v1/auth/login                    # Standard login
POST   /v1/auth/login/2fa                # Two-factor authentication
POST   /v1/auth/logout                   # Logout (invalidate token)
POST   /v1/auth/refresh                  # Refresh JWT token
POST   /v1/auth/register                 # User registration
POST   /v1/auth/forgot-password          # Password reset request
POST   /v1/auth/reset-password           # Password reset confirmation
GET    /v1/auth/me                       # Current user info
PUT    /v1/auth/me                       # Update current user
DELETE /v1/auth/me                       # Delete current user account
```

### 2. User Management
```
GET    /v1/users                         # List users (paginated)
GET    /v1/users/{id}                    # Get specific user
PUT    /v1/users/{id}                    # Update user (admin/self only)
DELETE /v1/users/{id}                    # Delete user (admin only)
GET    /v1/users/{id}/profile            # Public profile
PUT    /v1/users/{id}/profile            # Update profile
GET    /v1/users/{id}/presence           # User presence status
PUT    /v1/users/{id}/presence           # Update presence
```

### 3. Organizations & Workspaces
```
GET    /v1/organizations                 # List user's organizations
POST   /v1/organizations                 # Create organization
GET    /v1/organizations/{id}            # Get organization
PUT    /v1/organizations/{id}            # Update organization
DELETE /v1/organizations/{id}            # Delete organization
GET    /v1/organizations/{id}/members    # List members
POST   /v1/organizations/{id}/members    # Add member
DELETE /v1/organizations/{id}/members/{user_id}  # Remove member
```

### 4. Channels & Rooms
```
GET    /v1/channels                      # List accessible channels
POST   /v1/channels                      # Create channel
GET    /v1/channels/{id}                 # Get channel details
PUT    /v1/channels/{id}                 # Update channel
DELETE /v1/channels/{id}                 # Delete channel
GET    /v1/channels/{id}/members         # List channel members
POST   /v1/channels/{id}/members         # Add member to channel
DELETE /v1/channels/{id}/members/{user_id}  # Remove member
GET    /v1/channels/{id}/permissions     # Get channel permissions
PUT    /v1/channels/{id}/permissions     # Update permissions
```

### 5. Messaging
```
GET    /v1/channels/{id}/messages        # Get messages (paginated)
POST   /v1/channels/{id}/messages        # Send message
GET    /v1/messages/{id}                 # Get specific message
PUT    /v1/messages/{id}                 # Edit message
DELETE /v1/messages/{id}                 # Delete message
POST   /v1/messages/{id}/reactions       # Add reaction
DELETE /v1/messages/{id}/reactions/{emoji}  # Remove reaction
GET    /v1/messages/{id}/thread          # Get message thread
POST   /v1/messages/{id}/thread          # Reply to thread
```

### 6. Direct Messages
```
GET    /v1/dm/conversations              # List DM conversations
POST   /v1/dm/conversations              # Start new conversation
GET    /v1/dm/conversations/{id}         # Get conversation
GET    /v1/dm/conversations/{id}/messages  # Get DM messages
POST   /v1/dm/conversations/{id}/messages  # Send DM
```

### 7. File Management
```
GET    /v1/files                         # List files (paginated)
POST   /v1/files                         # Upload file
GET    /v1/files/{id}                    # Get file metadata
GET    /v1/files/{id}/download           # Download file
DELETE /v1/files/{id}                    # Delete file
GET    /v1/files/{id}/thumbnail          # Get file thumbnail
POST   /v1/files/{id}/share              # Generate share link
```

### 8. Voice & Video
```
POST   /v1/channels/{id}/voice/join      # Join voice channel
DELETE /v1/channels/{id}/voice/leave     # Leave voice channel
GET    /v1/channels/{id}/voice/participants  # List voice participants
POST   /v1/calls                         # Start video call
GET    /v1/calls/{id}                    # Get call details
POST   /v1/calls/{id}/join               # Join call
DELETE /v1/calls/{id}/leave              # Leave call
```

### 9. Administration
```
GET    /v1/admin/stats                   # System statistics
GET    /v1/admin/users                   # Admin user management
PUT    /v1/admin/users/{id}/status       # Update user status
GET    /v1/admin/audit-logs              # Audit logs
GET    /v1/admin/settings                # System settings
PUT    /v1/admin/settings                # Update settings
GET    /v1/admin/health                  # System health check
```

### 10. WebSocket Endpoints
```
WSS    /v1/ws/connect                    # Main WebSocket connection
WSS    /v1/ws/channels/{id}              # Channel-specific WebSocket
WSS    /v1/ws/dm/{conversation_id}       # DM WebSocket
WSS    /v1/ws/voice/{channel_id}         # Voice channel WebSocket
```

## Response Format Standards

### Success Response
```json
{
  "success": true,
  "data": { ... },
  "meta": {
    "timestamp": "2024-01-01T12:00:00Z",
    "request_id": "req_123456789"
  }
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid input provided",
    "details": {
      "field": "email",
      "reason": "Invalid email format"
    }
  },
  "meta": {
    "timestamp": "2024-01-01T12:00:00Z",
    "request_id": "req_123456789"
  }
}
```

### Pagination
```json
{
  "success": true,
  "data": [...],
  "pagination": {
    "page": 1,
    "per_page": 50,
    "total": 1250,
    "total_pages": 25,
    "has_next": true,
    "has_prev": false
  }
}
```

## HTTP Status Codes
- **200**: Success
- **201**: Created
- **204**: No Content (successful deletion)
- **400**: Bad Request (validation errors)
- **401**: Unauthorized (authentication required)
- **403**: Forbidden (insufficient permissions)
- **404**: Not Found
- **409**: Conflict (resource already exists)
- **422**: Unprocessable Entity (business logic error)
- **429**: Too Many Requests (rate limited)
- **500**: Internal Server Error
- **503**: Service Unavailable

## Rate Limiting
- **Authentication**: 5 requests per minute per IP
- **API Calls**: 1000 requests per hour per user
- **File Uploads**: 10 uploads per minute per user
- **WebSocket**: 100 messages per minute per user

## Security Headers
```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
X-XSS-Protection: 1; mode=block
Strict-Transport-Security: max-age=31536000; includeSubDomains
Content-Security-Policy: default-src 'self'
```

## Versioning Strategy
- **URL Versioning**: `/v1/`, `/v2/`, etc.
- **Backward Compatibility**: Maintain v1 for 2 years after v2 release
- **Deprecation Warnings**: Include deprecation headers
- **Migration Guides**: Provide clear upgrade paths
