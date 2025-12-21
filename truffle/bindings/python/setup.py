"""
Truffle Python Bindings (Native cgo)
"""
import os
import subprocess
import platform
from setuptools import setup, find_packages
from setuptools.command.build_py import build_py


class BuildGoLibrary(build_py):
    """Custom build command to compile Go shared library"""
    
    def run(self):
        """Build Go library before Python package"""
        print("Building Truffle Go library...")
        
        # Determine library extension
        system = platform.system()
        if system == "Darwin":
            lib_ext = "dylib"
        elif system == "Windows":
            lib_ext = "dll"
        else:
            lib_ext = "so"
        
        lib_name = f"libtruffle.{lib_ext}"
        lib_path = os.path.join("truffle", lib_name)
        
        # Check if Go is installed
        try:
            subprocess.run(["go", "version"], check=True, capture_output=True)
        except (subprocess.CalledProcessError, FileNotFoundError):
            raise RuntimeError(
                "Go compiler not found. Install Go from https://golang.org/dl/"
            )
        
        # Build Go shared library
        try:
            subprocess.run(
                ["go", "build", "-buildmode=c-shared", "-o", lib_name, "native.go"],
                cwd="truffle",
                check=True,
                capture_output=True,
                text=True
            )
            print(f"✅ Built {lib_path}")
        except subprocess.CalledProcessError as e:
            print(f"❌ Go build failed:")
            print(e.stdout)
            print(e.stderr)
            raise RuntimeError("Failed to build Go library") from e
        
        # Continue with normal Python build
        super().run()


with open("README.md", "r", encoding="utf-8") as fh:
    long_description = fh.read()

setup(
    name="truffle-aws",
    version="0.2.0",
    author="Your Name",
    author_email="you@example.com",
    description="Python bindings for Truffle AWS EC2 instance discovery (Native cgo)",
    long_description=long_description,
    long_description_content_type="text/markdown",
    url="https://github.com/yourusername/truffle",
    packages=find_packages(),
    cmdclass={
        'build_py': BuildGoLibrary,
    },
    package_data={
        'truffle': ['*.so', '*.dylib', '*.dll'],  # Include compiled libraries
    },
    include_package_data=True,
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Topic :: System :: Systems Administration",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
        "Programming Language :: Python :: 3.12",
    ],
    python_requires=">=3.8",
    install_requires=[
        # No Python dependencies - uses native Go library
    ],
    extras_require={
        "dev": [
            "pytest>=7.0",
            "black>=22.0",
            "mypy>=0.950",
            "ruff>=0.0.270",
        ],
    },
    keywords="aws ec2 instance-types spot-instances ml-capacity gpu cloud infrastructure cgo native",
    project_urls={
        "Bug Reports": "https://github.com/yourusername/truffle/issues",
        "Source": "https://github.com/yourusername/truffle",
        "Documentation": "https://github.com/yourusername/truffle/tree/main/bindings/python",
    },
    zip_safe=False,  # Required for including shared libraries
)

