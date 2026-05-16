[build-system]
requires = ["setuptools>=45", "wheel", "setuptools_scm[toml]>=6.2"]
build-backend = "setuptools.build_meta"

[project]
name = "hjtpx"
version = "1.0.0"
description = "hjtpx Python SDK - 极验行为验证系统 Python SDK"
readme = "README.md"
license = {text = "MIT"}
authors = [
    {name = "hjtpx Team", email = "team@hjtpx.com"}
]
keywords = ["captcha", "verification", "security", "hjtpx"]
classifiers = [
    "Development Status :: 5 - Production/Stable",
    "Intended Audience :: Developers",
    "License :: OSI Approved :: MIT License",
    "Operating System :: OS Independent",
    "Programming Language :: Python :: 3",
    "Programming Language :: Python :: 3.7",
    "Programming Language :: Python :: 3.8",
    "Programming Language :: Python :: 3.9",
    "Programming Language :: Python :: 3.10",
    "Programming Language :: Python :: 3.11",
    "Programming Language :: Python :: 3.12",
    "Topic :: Security",
    "Topic :: Software Development :: Libraries :: Python Modules",
]
requires-python = ">=3.7"
dependencies = []

[project.optional-dependencies]
dev = [
    "pytest>=7.0.0",
    "pytest-cov>=4.0.0",
    "pytest-asyncio>=0.21.0",
    "black>=23.0.0",
    "flake8>=6.0.0",
    "mypy>=1.0.0",
    "types-requests>=2.28.0",
]
test = [
    "pytest>=7.0.0",
    "pytest-cov>=4.0.0",
    "pytest-asyncio>=0.21.0",
    "responses>=0.23.0",
    "httpretty>=1.1.0",
]

[project.urls]
Homepage = "https://github.com/hjtpx/hjtpx"
Documentation = "https://github.com/hjtpx/hjtpx#readme"
Repository = "https://github.com/hjtpx/hjtpx.git"
Issues = "https://github.com/hjtpx/hjtpx/issues"

[tool.setuptools]
packages = ["hjtpx"]

[tool.setuptools.package-data]
hjtpx = ["py.typed"]

[tool.pytest.ini_options]
testpaths = ["tests"]
python_files = ["test_*.py"]
python_functions = ["test_*"]
addopts = "-v --tb=short"

[tool.black]
line-length = 100
target-version = ['py37', 'py38', 'py39', 'py310', 'py311']
include = '\.pyi?$'

[tool.mypy]
python_version = "3.7"
warn_return_any = true
warn_unused_configs = true
disallow_untyped_defs = false
