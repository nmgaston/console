# DMT Architecture with Redfish Integration

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              CLIENT APPLICATIONS                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Redfish    │  │   Console    │  │   WebUI      │  │  Automation  │   │
│  │   Clients    │  │     API      │  │   Client     │  │   Tools      │   │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘   │
└─────────┼──────────────────┼──────────────────┼──────────────────┼──────────┘
          │                  │                  │                  │
          │ HTTP/REST        │ HTTP/REST        │ WebSocket        │ HTTP/REST
          │                  │                  │                  │
┌─────────▼──────────────────▼──────────────────▼──────────────────▼──────────┐
│                         DMT CONSOLE SERVICE                                  │
│                                                                              │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                        HTTP/WebSocket Layer                            │ │
│  │  ┌──────────────────────┐              ┌──────────────────────┐       │ │
│  │  │  Redfish Controller  │              │  Console Controller  │       │ │
│  │  │  /redfish/v1/*       │              │  /api/v1/*          │       │ │
│  │  │                      │              │                      │       │ │
│  │  │ - ServiceRoot        │              │ - Devices            │       │ │
│  │  │ - Sessions           │              │ - AMT Explorer       │       │ │
│  │  │ - Systems            │              │ - Profiles           │       │ │
│  │  │ - ComputerSystem     │              │ - CIRA Configs       │       │ │
│  │  │ - Actions (Reset)    │              │ - WiFi Configs       │       │ │
│  │  │ - Boot Operations    │              │ - IEEE 802.1x        │       │ │
│  │  └──────────┬───────────┘              └──────────┬───────────┘       │ │
│  └─────────────┼──────────────────────────────────────┼──────────────────┘ │
│                │                                      │                     │
│  ┌─────────────▼──────────────────────────────────────▼──────────────────┐ │
│  │                           USE CASE LAYER                               │ │
│  │  ┌───────────────────┐              ┌────────────────────────────┐    │ │
│  │  │ Redfish Use Cases │              │   Console Use Cases        │    │ │
│  │  │                   │              │                            │    │ │
│  │  │ - Sessions Mgmt   │◄────────────►│ - Device Management        │    │ │
│  │  │ - System Info     │   Shared     │ - Alarms                   │    │ │
│  │  │ - Reset Actions   │   Device     │ - Certificates             │    │ │
│  │  │ - Boot Control    │   Context    │ - Connections              │    │ │
│  │  └─────────┬─────────┘              │ - Power Actions            │    │ │
│  │            │                        │ - Network Settings         │    │ │
│  │            │                        │ - KVM/VNC                  │    │ │
│  │            │                        └────────────┬───────────────┘    │ │
│  └────────────┼─────────────────────────────────────┼────────────────────┘ │
│               │                                     │                      │
│  ┌────────────▼─────────────────────────────────────▼────────────────────┐ │
│  │                         ENTITY/DOMAIN LAYER                            │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                │ │
│  │  │   Session    │  │    Device    │  │   Profile    │  ...            │ │
│  │  │   Entity     │  │   Entity     │  │   Entity     │                 │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘                │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
│                                     │                                       │
│  ┌──────────────────────────────────▼─────────────────────────────────────┐ │
│  │                      INFRASTRUCTURE LAYER                               │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                │ │
│  │  │   Database   │  │    Memory    │  │   External   │                │ │
│  │  │  Repository  │  │   Sessions   │  │     APIs     │                │ │
│  │  │  (Postgres)  │  │    Store     │  │              │                │ │
│  │  └──────────────┘  └──────────────┘  └──────────────┘                │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
└───────────────────────────────────┬──────────────────────────────────────────┘
                                    │
                                    │ AMT Commands/WSMAN
                                    │
                    │                                     │
                    │ AMT/WSMAN                         │ Redfish Protocol
                    │ (Console API)                     │ (Redfish API)
                    │                                   │
┌───────────────────▼───────────────┐   ┌───────────────▼──────────────────────┐
│    INTEL AMT DEVICES              │   │   REDFISH-COMPLIANT DEVICES          │
│  ┌──────────────┐                 │   │  ┌──────────────┐  ┌──────────────┐ │
│  │  AMT Device  │  ┌──────────┐  │   │  │  Dell iDRAC  │  │   HPE iLO    │ │
│  │   (vPro)     │  │   AMT    │  │   │  │              │  │              │ │
│  └──────────────┘  │  Device  │  │   │  └──────────────┘  └──────────────┘ │
│                    └──────────┘  │   │  ┌──────────────┐  ┌──────────────┐ │
│      (Console API Path)          │   │  │ Lenovo XCC   │  │  Cisco CIMC  │ │
└──────────────────────────────────┘   │  │              │  │              │ │
                                       │  └──────────────┘  └──────────────┘ │
                                       │  ┌──────────────┐  ┌──────────────┐ │
                                       │  │ Intel AMT    │  │    Other     │ │
                                       │  │ (Redfish)    │  │   Vendors    │ │
                                       │  └──────────────┘  └──────────────┘ │
                                       │      (Redfish API Path)              │
                                       └──────────────────────────────────────┘
```

## Key Integration Points

### 1. **Redfish API Layer** (`redfish/`)
- **Standards-Based**: DMTF Redfish specification compliance
- **OpenAPI Spec**: Auto-generated from DMTF schemas (`openapi/dmtf/`)
- **Controllers**: Handle HTTP requests for Redfish endpoints
- **Handlers**: Service root, sessions, systems, boot, reset actions

### 2. **Console API Layer** (`internal/controller/`)
- **Custom DMT APIs**: Intel AMT-specific operations
- **Device Management**: Full lifecycle management
- **Advanced Features**: CIRA, profiles, WiFi, 802.1x configurations

### 3. **Use Case Layer** (`internal/usecase/`, `redfish/internal/usecase/`)
- **Shared Business Logic**: Both APIs use same device context
- **Separation of Concerns**: Redfish use cases vs Console use cases
- **Common Device Access**: Single source of truth for device state

### 4. **Entity Layer** (`internal/entity/`, `redfish/internal/entity/`)
- **Domain Models**: Device, Profile, Session, etc.
- **Cross-API Compatibility**: Same entities used by both APIs

### 5. **Infrastructure Layer** (`internal/infrastructure/`)
- **Database**: PostgreSQL for persistent storage
- **In-Memory Store**: Sessions and temporary data
- **WSMAN Client**: Communication with Intel AMT devices (Console API)
- **Redfish Client**: HTTP/REST communication with Redfish endpoints (Redfish API)

## API Comparison

| Feature | Redfish API | Console API |
|---------|-------------|-------------|
| **Standard** | DMTF Redfish | Custom DMT |
| **Target** | Multi-vendor Redfish devices | Intel AMT-specific |
| **Devices** | Dell, HPE, Lenovo, Cisco, Intel AMT, etc. | Intel vPro/AMT only |
| **Protocol** | HTTP/REST (Redfish) | WSMAN/SOAP (Intel AMT) |
| **Use Cases** | Power, boot, reset, sessions, inventory | Profiles, CIRA, WiFi, KVM, 802.1x |
| **Discovery** | Standard metadata | Custom endpoints |
| **Auth** | Redfish sessions | JWT/Basic |
| **Integration** | Standard tools (any Redfish client) | Custom DMT clients |

## Benefits of Dual API Design

1. **Multi-Vendor Support**: Redfish enables management of Dell, HPE, Lenovo, Cisco, and other vendors
2. **Industry Compatibility**: Standard Redfish tools work out-of-the-box
3. **Intel AMT Deep Features**: Console APIs provide Intel-specific advanced features
4. **Flexibility**: Clients choose API based on device type and requirements
5. **Migration Path**: Gradual transition from proprietary to standard APIs
6. **Future-Proof**: Not locked into single vendor ecosystem

## Data Flow Example: Power Reset

### Redfish API Path (Multi-Vendor):
```
Redfish Client Request:
POST /redfish/v1/Systems/{id}/Actions/ComputerSystem.Reset
{"ResetType": "ForceRestart"}
          │
          ▼
Redfish Controller (HTTP handler)
          │
          ▼
Redfish Use Case (business logic)
          │
          ▼
Redfish Infrastructure (HTTP/REST to device)
          │
          ▼
Any Redfish Device (Dell iDRAC, HPE iLO, Intel AMT, etc.)
```

### Console API Path (Intel AMT Only):
```
Console Client Request:
POST /api/v1/devices/{id}/power/actions/reset
          │
          ▼
Console Controller (HTTP handler)
          │
          ▼
Console Use Case (business logic)
          │
          ▼
Device Entity (Intel AMT specific)
          │
          ▼
WSMAN Infrastructure (SOAP/WSMAN to device)
          │
          ▼
Intel AMT Device (execute reset via AMT commands)
```

**Key Difference**: 
- **Redfish API** = Standard HTTP/REST to any Redfish-compliant device
- **Console API** = Proprietary WSMAN/SOAP to Intel AMT devices only
