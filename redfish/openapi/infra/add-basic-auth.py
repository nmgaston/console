#!/usr/bin/env python3

import yaml
import os
import subprocess
import sys

def add_basic_auth_to_existing_spec():
      
    # Path to the existing merged OpenAPI file
    spec_file = '../merged/redfish-openapi.yaml'
    
    if not os.path.exists(spec_file):
        print(f"Error: {spec_file} does not exist")
        return False
    
    print(f"Loading existing OpenAPI spec from {spec_file}...")
    
    # Load the existing spec
    with open(spec_file, 'r') as f:
        spec = yaml.safe_load(f)
    
    print("Adding Basic Authentication configuration...")
    
    # Ensure components and securitySchemes exist
    if 'components' not in spec:
        spec['components'] = {}
    if 'securitySchemes' not in spec['components']:
        spec['components']['securitySchemes'] = {}
    
    # Add BasicAuth security scheme (only if not already present)
    if 'BasicAuth' not in spec['components']['securitySchemes']:
        spec['components']['securitySchemes']['BasicAuth'] = {
            'type': 'http',
            'scheme': 'basic',
            'description': 'HTTP Basic Authentication for Redfish API'
        }
        print("  Added BasicAuth security scheme")
    else:
        print("  BasicAuth security scheme already exists")
    
    # Add global security requirements (only if not already present)
    if 'security' not in spec:
        spec['security'] = []
    
    # Check if BasicAuth is already in global security
    has_basic_auth = any('BasicAuth' in req for req in spec['security'] if isinstance(req, dict))
    has_empty_auth = {} in spec['security']
    
    if not has_basic_auth:
        spec['security'].append({'BasicAuth': []})
        print("  Added BasicAuth to global security")
    
    if not has_empty_auth:
        spec['security'].append({})
        print("  Added empty auth option for public endpoints")
    
    # Configure endpoint security (only if not already configured)
    if 'paths' in spec:
        # Public endpoints (Redfish spec compliance)
        public_endpoints = ['/redfish/v1/', '/redfish/v1/$metadata']
        
        for endpoint in public_endpoints:
            if endpoint in spec['paths']:
                for method in spec['paths'][endpoint]:
                    if method.lower() in ['get', 'head', 'options']:
                        if 'security' not in spec['paths'][endpoint][method]:
                            spec['paths'][endpoint][method]['security'] = [{}]
                            print(f"  Set {endpoint} {method.upper()} as public (no auth)")
        
        # Add auth requirement to protected endpoints (only if not already configured)
        protected_count = 0
        for path, path_spec in spec['paths'].items():
            if path not in public_endpoints:
                for method, method_spec in path_spec.items():
                    if method.lower() in ['get', 'post', 'put', 'patch', 'delete']:
                        if 'security' not in method_spec:
                            method_spec['security'] = [{'BasicAuth': []}]
                            protected_count += 1
        
        if protected_count > 0:
            print(f"  Added BasicAuth requirement to {protected_count} protected endpoints")
    
    # Add global Basic Auth metadata (only if not already present)
    if 'x-basic-auth' not in spec:
        spec['x-basic-auth'] = {
            'default-auth': 'BasicAuth',
            'redfish-compliant': True,
            'service-root-public': True
        }
        print("  Added Basic Auth metadata")
    else:
        print("  Basic Auth metadata already exists")
    
    # Write the enhanced spec back to the file
    with open(spec_file, 'w') as f:
        yaml.dump(spec, f, default_flow_style=False, sort_keys=False)
    
    print(f"Basic Auth configuration added to {spec_file}")
    return True

def regenerate_go_code():
    print("Regenerating Go server code...")
    
    try:
        result = subprocess.run(['make', 'rf-generate'], 
                              capture_output=True, text=True, cwd='.')
        
        if result.returncode == 0:
            print("Go server code regenerated!")
            return True
        else:
            print(f"Code generation failed: {result.stderr}")
            return False
            
    except Exception as e:
        print(f"Error: {e}")
        return False

def verify_auth_implementation():

    middleware_file = '../../internal/controller/http/v1/handler/middleware.go'
    
    if os.path.exists(middleware_file):
        print(f"Authentication middleware found: {middleware_file}")
        return True
    else:
        print(f"Authentication middleware missing: {middleware_file}")
        return False

if __name__ == "__main__":
    print("Setting up Basic Authentication for Redfish API...")
    
    # Step 1: Update OpenAPI spec
    if not add_basic_auth_to_existing_spec():
        print("Failed to update OpenAPI specification")
        sys.exit(1)
    
    # Step 2: Verify auth implementation exists  
    if not verify_auth_implementation():
        print("Authentication implementation missing")
        sys.exit(1)
    
    # Step 3: Regenerate Go code
    if not regenerate_go_code():
        print("Failed to regenerate Go code")
        sys.exit(1)
