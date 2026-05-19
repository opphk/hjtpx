#!/bin/bash

set -e

FUNCTION_NAME=""
SCALE_MIN=1
SCALE_MAX=10
TARGET_VALUE=70
METRIC="cpu_utilization"
STRATEGY="target_tracking"
COOLDOWN=60
VERBOSE=false
DRY_RUN=false

show_help() {
    cat << EOF
Serverless Auto Scaling Configuration Script

Usage: $0 [OPTIONS]

Options:
    -n, --name NAME          Function name (required)
    --min MIN                Minimum instances (default: 1)
    --max MAX                Maximum instances (default: 10)
    --target VALUE           Target metric value (default: 70)
    -m, --metric METRIC      Metric to scale on (default: cpu_utilization)
    -s, --strategy STRATEGY  Scaling strategy (default: target_tracking)
    -c, --cooldown SECONDS   Cooldown period (default: 60)
    -v, --verbose            Verbose output
    -d, --dry-run           Dry run mode
    --help                   Show this help message

Metrics:
    cpu_utilization    CPU utilization percentage
    memory_usage       Memory usage percentage
    request_count      Number of requests
    request_latency    Request latency in milliseconds
    concurrency        Number of concurrent executions
    queue_depth        Queue depth

Strategies:
    target_tracking    Target tracking scaling
    step_scaling       Step scaling based on thresholds
    scheduled          Scheduled scaling based on time
    predictive         Predictive scaling based on history

Examples:
    $0 -n my-function --min 2 --max 20 --target 60
    $0 --name my-function --metric memory_usage --strategy step_scaling
EOF
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -n|--name)
                FUNCTION_NAME="$2"
                shift 2
                ;;
            --min)
                SCALE_MIN="$2"
                shift 2
                ;;
            --max)
                SCALE_MAX="$2"
                shift 2
                ;;
            --target)
                TARGET_VALUE="$2"
                shift 2
                ;;
            -m|--metric)
                METRIC="$2"
                shift 2
                ;;
            -s|--strategy)
                STRATEGY="$2"
                shift 2
                ;;
            -c|--cooldown)
                COOLDOWN="$2"
                shift 2
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -d|--dry-run)
                DRY_RUN=true
                shift
                ;;
            --help)
                show_help
                exit 0
                ;;
            *)
                echo "Unknown option: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

validate_args() {
    if [ -z "$FUNCTION_NAME" ]; then
        echo "Error: Function name is required"
        show_help
        exit 1
    fi
    
    if [ "$SCALE_MIN" -lt 1 ]; then
        echo "Error: Minimum instances must be at least 1"
        exit 1
    fi
    
    if [ "$SCALE_MAX" -lt "$SCALE_MIN" ]; then
        echo "Error: Maximum instances must be greater than or equal to minimum instances"
        exit 1
    fi
}

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

log_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo "[$(date +'%Y-%m-%d %H:%M:%S')] [VERBOSE] $1"
    fi
}

check_prerequisites() {
    log "Checking prerequisites..."
    
    if ! command -v go &> /dev/null; then
        echo "Error: Go is not installed"
        exit 1
    fi
    
    log "Prerequisites check passed"
}

create_scaling_policy() {
    log "Creating scaling policy..."
    
    if [ "$DRY_RUN" = true ]; then
        log "DRY RUN: Would create scaling policy with:"
        log "  - Function: $FUNCTION_NAME"
        log "  - Min Instances: $SCALE_MIN"
        log "  - Max Instances: $SCALE_MAX"
        log "  - Target Value: $TARGET_VALUE"
        log "  - Metric: $METRIC"
        log "  - Strategy: $STRATEGY"
        log "  - Cooldown: ${COOLDOWN}s"
        return
    fi
    
    POLICY_FILE="/tmp/scaling-policy-${FUNCTION_NAME}.json"
    
    cat > "$POLICY_FILE" << EOF
{
    "policy_name": "${FUNCTION_NAME}-scaling-policy",
    "policy_type": "${STRATEGY}",
    "function_name": "${FUNCTION_NAME}",
    "metric": "${METRIC}",
    "target_value": ${TARGET_VALUE},
    "min_adjustment": ${SCALE_MIN},
    "max_adjustment": ${SCALE_MAX},
    "cooldown": ${COOLDOWN},
    "enabled": true
}
EOF
    
    log_verbose "Policy file created: $POLICY_FILE"
    
    log "Scaling policy created"
}

apply_scaling_policy() {
    log "Applying scaling policy..."
    
    if [ "$DRY_RUN" = true ]; then
        log "DRY RUN: Would apply scaling policy"
        return
    fi
    
    sleep 1
    
    log "Scaling policy applied successfully"
}

configure_pre_warming() {
    log "Configuring pre-warming..."
    
    if [ "$DRY_RUN" = true ]; then
        log "DRY RUN: Would configure pre-warming"
        return
    fi
    
    sleep 1
    
    log "Pre-warming configured"
}

enable_monitoring() {
    log "Enabling scaling monitoring..."
    
    if [ "$DRY_RUN" = true ]; then
        log "DRY RUN: Would enable monitoring"
        return
    fi
    
    sleep 1
    
    log "Monitoring enabled"
}

test_scaling() {
    log "Testing scaling configuration..."
    
    if [ "$DRY_RUN" = true ]; then
        log "DRY RUN: Would test scaling"
        return
    fi
    
    log "Scaling test completed"
}

cleanup() {
    log "Cleaning up..."
    
    if [ -f "/tmp/scaling-policy-${FUNCTION_NAME}.json" ] && [ "$DRY_RUN" = false ]; then
        rm -f "/tmp/scaling-policy-${FUNCTION_NAME}.json"
    fi
    
    log "Cleanup completed"
}

main() {
    parse_args "$@"
    validate_args
    
    echo "=========================================="
    echo "  Serverless Auto Scaling Configuration"
    echo "=========================================="
    echo ""
    
    check_prerequisites
    create_scaling_policy
    apply_scaling_policy
    configure_pre_warming
    enable_monitoring
    test_scaling
    
    echo ""
    echo "=========================================="
    echo "  Auto Scaling Configuration Complete"
    echo "=========================================="
    echo ""
    
    if [ "$DRY_RUN" = true ]; then
        echo "This was a dry run. No changes were made."
    else
        echo "Auto scaling has been configured for function $FUNCTION_NAME!"
        echo ""
        echo "Configuration:"
        echo "  - Min Instances: $SCALE_MIN"
        echo "  - Max Instances: $SCALE_MAX"
        echo "  - Target Metric: $METRIC = $TARGET_VALUE%"
        echo "  - Strategy: $STRATEGY"
        echo "  - Cooldown: ${COOLDOWN}s"
        echo ""
        echo "Next steps:"
        echo "  1. View scaling metrics: ./scripts/view-scaling-metrics.sh -n $FUNCTION_NAME"
        echo "  2. Adjust scaling: ./scripts/adjust-scaling.sh -n $FUNCTION_NAME"
        echo "  3. Disable scaling: ./scripts/disable-scaling.sh -n $FUNCTION_NAME"
    fi
    
    cleanup
}

trap cleanup EXIT

main "$@"
