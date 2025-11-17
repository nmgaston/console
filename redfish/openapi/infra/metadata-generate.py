#!/usr/bin/env python3
"""Generate Redfish metadata.xml from DMTF schemas."""

import os
import sys
import re
import yaml
from typing import List, Dict, Tuple, Optional


def extract_schema_info(yaml_filename: str) -> Optional[Dict]:
    """Extract schema name and version from YAML filename."""
    match = re.match(r'(\w+)(\.v(\d+_\d+_\d+))?\.yaml$', yaml_filename)
    if not match:
        return None
    
    return {
        "name": match.group(1),
        "version": match.group(3),
    }


def extract_versioned_namespaces(yaml_path: str) -> List[str]:
    """Extract version-specific namespaces from YAML schema definitions."""
    versioned = set()
    
    try:
        with open(yaml_path, 'r') as f:
            content = yaml.safe_load(f)
        
        if not content or 'components' not in content:
            return []
        
        schemas = content.get('components', {}).get('schemas', {})
        
        for schema_name in schemas.keys():
            match = re.search(r'(\w+)_v(\d+_\d+_\d+)_', schema_name)
            if match:
                versioned.add(match.group(2))
    
    except Exception as e:
        print(f"⚠️  Warning parsing {yaml_path}: {e}")
    
    return sorted(list(versioned))


def discover_schemas() -> Dict[str, Dict]:
    """Auto-discover schemas from YAML files in dmtf/ directory."""
    dmtf_dir = os.path.join(
        os.path.dirname(__file__),
        "..",
        "dmtf"
    )
    
    discovered = {}
    
    if not os.path.exists(dmtf_dir):
        print(f"⚠️  DMTF directory not found: {dmtf_dir}")
        return discovered
    
    try:
        print(f"📂 Scanning DMTF directory: {dmtf_dir}")
        
        for yaml_filename in sorted(os.listdir(dmtf_dir)):
            if not yaml_filename.endswith(('.yaml', '.yml')):
                continue
            
            schema_info = extract_schema_info(yaml_filename)
            if not schema_info:
                continue
            
            schema_name = schema_info["name"]
            file_version = schema_info["version"]
            xml_filename = f"{schema_name}_v1.xml"
            
            if xml_filename in discovered and not file_version:
                continue
            
            yaml_path = os.path.join(dmtf_dir, yaml_filename)
            namespaces = []
            namespaces.append((schema_name, None))
            
            versioned = extract_versioned_namespaces(yaml_path)
            for ver in versioned:
                namespaces.append((schema_name, ver))
            
            if schema_name == "RedfishExtensions":
                namespaces = [("RedfishExtensions", "v1_0_0", "Redfish")]
            
            discovered[xml_filename] = {
                "namespaces": namespaces,
                "source_file": yaml_filename
            }
            
            print(f"  ✓ {yaml_filename} -> {xml_filename}")
        
        print(f"✓ Discovered {len(discovered)} schemas")
        return discovered
        
    except Exception as e:
        print(f"✗ Error scanning DMTF directory: {e}")
        return discovered


def generate_schema_references(schemas_info: Dict[str, Dict]) -> List[str]:
    """Generate EDMX reference entries for discovered schemas."""
    references = []
    
    for xml_filename in sorted(schemas_info.keys()):
        schema_data = schemas_info[xml_filename]
        namespaces = schema_data.get("namespaces", [])
        
        references.append(f'    <edmx:Reference Uri="http://redfish.dmtf.org/schemas/v1/{xml_filename}">')
        
        for ns_tuple in namespaces:
            if len(ns_tuple) == 3:
                ns_name, ns_version, alias = ns_tuple
                full_namespace = f"{ns_name}.{ns_version}" if ns_version else ns_name
                references.append(f'        <edmx:Include Namespace="{full_namespace}" Alias="{alias}"/>')
            elif len(ns_tuple) == 2:
                ns_name, ns_version = ns_tuple
                full_namespace = f"{ns_name}.{ns_version}" if ns_version else ns_name
                references.append(f'        <edmx:Include Namespace="{full_namespace}"/>')
        
        references.append(f'    </edmx:Reference>')
    
    return references


def generate_metadata_xml(discovered_schemas: Dict[str, Dict]) -> bool:
    """Generate metadata.xml reference file from discovered schemas."""
    metadata_path = os.path.join(
        os.path.dirname(__file__),
        "..",
        "..",
        "internal",
        "controller",
        "http",
        "v1",
        "generated",
        "metadata.xml"
    )
    
    try:
        print(f"�� Generating metadata.xml...")
        
        if not discovered_schemas:
            print("⚠️  No schemas discovered")
            return False
        
        references = generate_schema_references(discovered_schemas)
        
        metadata_content = '''<?xml version="1.0" encoding="UTF-8"?>
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="4.0">
''' + '\n'.join(references) + '''
    <edmx:DataServices>
        <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="Service">
            <EntityContainer Name="Service" Extends="ServiceRoot.v1_19_0.ServiceContainer"/>
        </Schema>
    </edmx:DataServices>
</edmx:Edmx>
'''
        
        os.makedirs(os.path.dirname(metadata_path), exist_ok=True)
        
        with open(metadata_path, 'w') as f:
            f.write(metadata_content)
        
        print(f"✓ Generated {metadata_path}")
        print(f"✓ Contains {len(discovered_schemas)} schemas")
        return True
        
    except Exception as e:
        print(f"✗ Failed to generate metadata.xml: {e}")
        return False


def main() -> int:
    """Main entry point for metadata generation."""
    print("\n" + "=" * 60)
    print("Metadata Generator - DMTF Redfish Schemas (Auto-Discovery)")
    print("=" * 60)
    
    discovered_schemas = discover_schemas()
    
    if not discovered_schemas:
        print("❌ No schemas discovered")
        return 1
    
    if not generate_metadata_xml(discovered_schemas):
        return 1
    
    print("\n✓ Metadata generation completed successfully!")
    return 0


if __name__ == "__main__":
    sys.exit(main())
