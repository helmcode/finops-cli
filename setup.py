from setuptools import setup, find_packages

with open('README.md', 'r', encoding='utf-8') as f:
    long_description = f.read()

setup(
    name='finops-cli',
    version='0.1.0',
    author='Cristian Córdova',
    author_email='cristian+finops-cli@helmcode.com',
    description='FinOps CLI - Tool for analyzing AWS costs.',
    long_description=long_description,
    long_description_content_type='text/markdown',
    url='https://github.com/helmcode/finops-cli',
    packages=find_packages(include=['cli', 'cli.src']),
    include_package_data=True,
    install_requires=[
        'boto3>=1.20.0',
        'click>=8.0.0',
        'tabulate>=0.8.9',
        'python-dotenv>=0.19.0',
    ],
    entry_points={
        'console_scripts': [
            'finops=cli.finops_cli:cli',
        ],
    },
    python_requires='>=3.7',
    classifiers=[
        'Development Status :: 4 - Beta',
        'Intended Audience :: Developers',
        'Intended Audience :: System Administrators',
        'License :: OSI Approved :: MIT License',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
        'Programming Language :: Python :: 3.9',
        'Programming Language :: Python :: 3.10',
        'Operating System :: OS Independent',
        'Topic :: System :: Systems Administration',
        'Topic :: Utilities',
    ],
    package_data={
        'cli': ['src/*.py'],
    }
)
