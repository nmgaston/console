#!/usr/bin/env python3
"""
Generate Redfish OData CSDL metadata.xml from DMTF schemas.

This script generates metadata.xml with references to DMTF Redfish schemas.

Usage:
    python3 metadata-generate.py
"""

import os
import sys
from typing import List

# Core CSDL schemas needed for basic Redfish service
# Currently focused on: Systems metadata and Power management
CORE_SCHEMAS = [
    "RedfishExtensions_v1.xml",
    "Resource_v1.xml",
    "ServiceRoot_v1.xml",
    "ComputerSystem_v1.xml",
    "ComputerSystemCollection_v1.xml",
    "Task_v1.xml",
    "Message_v1.xml",
]


def generate_metadata_xml(schemas: List[str]) -> bool:
    """Generate metadata.xml reference file based on schemas."""
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
        print(f"📝 Generating metadata.xml...")
        
        # Generate EDMX references for each schema
        references = []
        for schema in sorted(schemas):
            if schema.endswith(".xml"):
                # Extract namespace from filename
                # Example: ServiceRoot_v1.xml -> ServiceRoot
                namespace = schema.replace("_v1.xml", "").replace("_", ".")
                
                references.append(f'    <edmx:Reference Uri="http://redfish.dmtf.org/schemas/v1/{schema}">')
                references.append(f'        <edmx:Include Namespace="{namespace}"/>')
                references.append(f'    </edmx:Reference>')
        
        # Build complete metadata XML
        metadata_content = '''<?xml version="1.0" encoding="UTF-8"?>
<!-- This file is auto-generated from DMTF Redfish schemas. -->
<!-- Source: https://github.com/DMTF/Redfish-Publications -->
<!-- To regenerate: python3 redfish/openapi/infra/metadata-generate.py -->
<edmx:Edmx xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx" Version="4.0">
''' + '\n'.join(references) + '''
    <edmx:DataServices>
        <Schema xmlns="http://docs.oasis-open.org/odata/ns/edm" Namespace="Service">
            <EntityContainer Name="Service" Extends="ServiceRoot.v1_19_0.ServiceContainer"/>
        </Schema>
    </edmx:DataServices>
</edmx:Edmx>
'''
        
        # Ensure directory exists
        os.makedirs(os.path.dirname(metadata_path), exist_ok=True)
        
        # Write metadata file
        with open(metadata_path, 'w') as f:
            f.write(metadata_content)
        
        print(f"✓ Generated {metadata_path}")
        print(f"✓ Contains references to {len(references)//3} schemas")
        return True
        
    except Exception as e:
        print(f"✗ Failed to generate metadata.xml: {e}")
        return False


def main() -> int:
    """Main entry point."""
    print("\n" + "=" * 50)
    print("Metadata Generator - DMTF Redfish Schemas")
    print("=" * 50)
    
    if not generate_metadata_xml(CORE_SCHEMAS):
        return 1
    
    print("\n✓ Metadata generation completed successfully!")
    return 0


if __name__ == "__main__":
    sys.exit(main())
