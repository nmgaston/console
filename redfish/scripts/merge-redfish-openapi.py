#!/usr/bin/env python3
import yaml
import os
import glob
import re
from pathlib import Path

def remove_version_from_schema_name(schema_name):
    """Remove version information from schema name (e.g., Message_v1_2_1_Message -> Message_Message)"""
    # Pattern to match version info like _v1_2_1_ or _v1_0_0_
    version_pattern = r'_v\d+_\d+_\d+_'
    return re.sub(version_pattern, '_', schema_name)

def convert_file_refs_to_internal(obj):
    """Recursively convert file references to internal schema references and clean schema names"""
    if isinstance(obj, dict):
        new_obj = {}
        for key, value in obj.items():
            if key == '$ref' and isinstance(value, str):
                # Convert file references like "Resource.yaml#/components/schemas/Resource_Health"
                # to internal references like "#/components/schemas/Resource_Health"
                if '.yaml#/components/schemas/' in value:
                    schema_name = value.split('#/components/schemas/')[-1]
                    # Clean the schema name by removing version info
                    clean_schema_name = remove_version_from_schema_name(schema_name)
                    new_obj[key] = f"#/components/schemas/{clean_schema_name}"
                elif value.startswith('#/components/schemas/'):
                    # Also clean internal references
                    schema_name = value.split('#/components/schemas/')[-1]
                    clean_schema_name = remove_version_from_schema_name(schema_name)
                    new_obj[key] = f"#/components/schemas/{clean_schema_name}"
                else:
                    new_obj[key] = value
            else:
                new_obj[key] = convert_file_refs_to_internal(value)
        return new_obj
    elif isinstance(obj, list):
        return [convert_file_refs_to_internal(item) for item in obj]
    else:
        return obj

def merge_openapi_files():
    # Load the main OpenAPI file
    main_openapi_path = 'openapi/dmtf/openapi.yaml'

    if not os.path.exists(main_openapi_path):
        # If main openapi.yaml doesn't exist, create a basic structure
        print(f"Main OpenAPI file not found at {main_openapi_path}, creating basic structure...")
        main_spec = {
            'openapi': '3.0.0',
            'info': {
                'title': 'Redfish API',
                'version': '1.0.0',
                'description': 'Merged Redfish API specification'
            },
            'paths': {},
            'components': {
                'schemas': {}
            }
        }
    else:
        with open(main_openapi_path, 'r') as f:
            main_spec = yaml.safe_load(f)

    # Ensure components and schemas exist
    if 'components' not in main_spec:
        main_spec['components'] = {}
    if 'schemas' not in main_spec['components']:
        main_spec['components']['schemas'] = {}

    # Find all YAML schema files in the openapi/dmtf directory
    yaml_files = glob.glob('openapi/dmtf/*.yaml')
    yaml_files = [f for f in yaml_files if f != main_openapi_path]

    print(f"Found {len(yaml_files)} schema files to merge:")
    for f in yaml_files:
        print(f"  - {f}")

    # Merge schemas from all files
    for yaml_file in yaml_files:
        try:
            with open(yaml_file, 'r') as f:
                schema_spec = yaml.safe_load(f)

            # Extract schemas from the file
            if 'components' in schema_spec and 'schemas' in schema_spec['components']:
                schemas = schema_spec['components']['schemas']
                print(f"Merging {len(schemas)} schemas from {yaml_file}")

                # Add schemas to main spec with version prioritization
                for schema_name, schema_def in schemas.items():
                    # Remove version info from schema name for conflict checking
                    clean_schema_name = remove_version_from_schema_name(schema_name)

                    # Check if we have a version in the current schema name
                    has_version = '_v' in schema_name and clean_schema_name != schema_name

                    # Check for conflicts
                    if clean_schema_name in main_spec['components']['schemas']:
                        if has_version:
                            # Current schema has version, replace the existing non-versioned one
                            print(f"  Replacing existing {clean_schema_name} with versioned {schema_name}")
                            main_spec['components']['schemas'][clean_schema_name] = schema_def
                        else:
                            # Current schema has no version, skip it in favor of existing
                            print(f"  Skipping non-versioned {schema_name} - keeping existing {clean_schema_name}")
                    else:
                        # No conflict - use the clean name
                        print(f"  Adding schema: {schema_name} -> {clean_schema_name}")
                        main_spec['components']['schemas'][clean_schema_name] = schema_def

            # Extract paths from the file
            if 'paths' in schema_spec:
                if 'paths' not in main_spec:
                    main_spec['paths'] = {}
                print(f"Merging {len(schema_spec['paths'])} paths from {yaml_file}")
                main_spec['paths'].update(schema_spec['paths'])

        except Exception as e:
            print(f"Error processing {yaml_file}: {e}")
            continue

    # Convert all file references to internal references
    print("Converting file references to internal references...")
    main_spec = convert_file_refs_to_internal(main_spec)

    # Remove duplicate /redfish/v1/ path to avoid Go method conflicts
    if 'paths' in main_spec:
        if '/redfish/v1/' in main_spec['paths'] and '/redfish/v1' in main_spec['paths']:
            print("Removing duplicate /redfish/v1/ path...")
            del main_spec['paths']['/redfish/v1/']

    # Create output directory if it doesn't exist
    os.makedirs('openapi/merged', exist_ok=True)

    # Write the merged file
    output_path = 'openapi/merged/redfish-openapi.yaml'
    with open(output_path, 'w') as f:
        yaml.dump(main_spec, f, default_flow_style=False, sort_keys=False)

    print(f"Merged OpenAPI spec written to {output_path}")
    print(f"Total schemas in merged file: {len(main_spec['components']['schemas'])}")
    if 'paths' in main_spec:
        print(f"Total paths in merged file: {len(main_spec['paths'])}")

if __name__ == "__main__":
    merge_openapi_files()