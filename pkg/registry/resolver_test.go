package registry

import (
	"errors"
	"reflect"
	"testing"

	"github.com/openshift/ci-tools/pkg/api"
	"k8s.io/apimachinery/pkg/util/diff"
)

func TestResolve(t *testing.T) {
	reference1 := "generic-unit-test"
	teardownRef := "teardown"
	fipsPreChain := "install-fips"
	nestedChains := "nested-chains"
	chainInstall := "install-chain"
	awsWorkflow := "ipi-aws"
	for _, testCase := range []struct {
		name        string
		config      api.MultiStageTestConfiguration
		stepMap     ReferenceByName
		chainMap    ChainByName
		workflowMap WorkflowByName
		expectedRes api.MultiStageTestConfigurationLiteral
		expectedErr error
	}{{
		// This is a full config that should not change (other than struct) when passed to the Resolver
		name: "Full AWS IPI",
		config: api.MultiStageTestConfiguration{
			ClusterProfile: api.ClusterProfileAWS,
			Pre: []api.TestStep{{
				LiteralTestStep: &api.LiteralTestStep{
					As:       "ipi-install",
					From:     "installer",
					Commands: "openshift-cluster install",
					Resources: api.ResourceRequirements{
						Requests: api.ResourceList{"cpu": "1000m"},
						Limits:   api.ResourceList{"memory": "2Gi"},
					}},
			}},
			Test: []api.TestStep{{
				LiteralTestStep: &api.LiteralTestStep{
					As:       "e2e",
					From:     "my-image",
					Commands: "make custom-e2e",
					Resources: api.ResourceRequirements{
						Requests: api.ResourceList{"cpu": "1000m"},
						Limits:   api.ResourceList{"memory": "2Gi"},
					}},
			}},
			Post: []api.TestStep{{
				LiteralTestStep: &api.LiteralTestStep{
					As:       "ipi-teardown",
					From:     "installer",
					Commands: "openshift-cluster destroy",
					Resources: api.ResourceRequirements{
						Requests: api.ResourceList{"cpu": "1000m"},
						Limits:   api.ResourceList{"memory": "2Gi"},
					}},
			}},
		},
		expectedRes: api.MultiStageTestConfigurationLiteral{
			ClusterProfile: api.ClusterProfileAWS,
			Pre: []api.LiteralTestStep{{
				As:       "ipi-install",
				From:     "installer",
				Commands: "openshift-cluster install",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
			Test: []api.LiteralTestStep{{
				As:       "e2e",
				From:     "my-image",
				Commands: "make custom-e2e",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
			Post: []api.LiteralTestStep{{
				As:       "ipi-teardown",
				From:     "installer",
				Commands: "openshift-cluster destroy",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
		},
	}, {
		name: "Test with reference",
		config: api.MultiStageTestConfiguration{
			ClusterProfile: api.ClusterProfileAWS,
			Test: []api.TestStep{{
				Reference: &reference1,
			}},
		},
		stepMap: ReferenceByName{
			reference1: {
				As:       "generic-unit-test",
				From:     "my-image",
				Commands: "make test/unit",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			},
		},
		expectedRes: api.MultiStageTestConfigurationLiteral{
			ClusterProfile: api.ClusterProfileAWS,
			Test: []api.LiteralTestStep{{
				As:       "generic-unit-test",
				From:     "my-image",
				Commands: "make test/unit",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
		},
	}, {
		name: "Test with broken reference",
		config: api.MultiStageTestConfiguration{
			ClusterProfile: api.ClusterProfileAWS,
			Test: []api.TestStep{{
				Reference: &reference1,
			}},
		},
		stepMap: ReferenceByName{
			"generic-unit-test-2": {
				As:       "generic-unit-test-2",
				From:     "my-image",
				Commands: "make test/unit",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			},
		},
		expectedRes: api.MultiStageTestConfigurationLiteral{},
		expectedErr: errors.New("test: invalid step reference: generic-unit-test"),
	}, {
		name: "Test with chain and reference",
		config: api.MultiStageTestConfiguration{
			ClusterProfile: api.ClusterProfileAWS,
			Pre: []api.TestStep{{
				Chain: &fipsPreChain,
			}},
			Test: []api.TestStep{{
				LiteralTestStep: &api.LiteralTestStep{
					As:       "e2e",
					From:     "my-image",
					Commands: "make custom-e2e",
					Resources: api.ResourceRequirements{
						Requests: api.ResourceList{"cpu": "1000m"},
						Limits:   api.ResourceList{"memory": "2Gi"},
					}},
			}},
			Post: []api.TestStep{{
				Reference: &teardownRef,
			}},
		},
		chainMap: ChainByName{
			fipsPreChain: {
				Steps: []api.TestStep{{
					LiteralTestStep: &api.LiteralTestStep{
						As:       "ipi-install",
						From:     "installer",
						Commands: "openshift-cluster install",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}, {
					LiteralTestStep: &api.LiteralTestStep{
						As:       "enable-fips",
						From:     "fips-enabler",
						Commands: "enable_fips",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
			},
		},
		stepMap: ReferenceByName{
			teardownRef: {
				As:       "ipi-teardown",
				From:     "installer",
				Commands: "openshift-cluster destroy",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				}},
		},
		expectedRes: api.MultiStageTestConfigurationLiteral{
			ClusterProfile: api.ClusterProfileAWS,
			Pre: []api.LiteralTestStep{{
				As:       "ipi-install",
				From:     "installer",
				Commands: "openshift-cluster install",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}, {
				As:       "enable-fips",
				From:     "fips-enabler",
				Commands: "enable_fips",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
			Test: []api.LiteralTestStep{{
				As:       "e2e",
				From:     "my-image",
				Commands: "make custom-e2e",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
			Post: []api.LiteralTestStep{{
				As:       "ipi-teardown",
				From:     "installer",
				Commands: "openshift-cluster destroy",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
		},
	}, {
		name: "Test with broken chain",
		config: api.MultiStageTestConfiguration{
			ClusterProfile: api.ClusterProfileAWS,
			Test: []api.TestStep{{
				Reference: &fipsPreChain,
			}},
		},
		chainMap: ChainByName{
			"broken": {
				Steps: []api.TestStep{{
					LiteralTestStep: &api.LiteralTestStep{
						As:       "generic-unit-test-2",
						From:     "my-image",
						Commands: "make test/unit",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
			},
		},
		expectedRes: api.MultiStageTestConfigurationLiteral{},
		expectedErr: errors.New("test: invalid step reference: install-fips"),
	}, {
		name: "Test with nested chains",
		config: api.MultiStageTestConfiguration{
			ClusterProfile: api.ClusterProfileAWS,
			Pre: []api.TestStep{{
				Chain: &nestedChains,
			}},
		},
		chainMap: ChainByName{
			nestedChains: {
				Steps: []api.TestStep{{
					Chain: &chainInstall,
				}, {
					LiteralTestStep: &api.LiteralTestStep{
						As:       "enable-fips",
						From:     "fips-enabler",
						Commands: "enable_fips",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
			},
			chainInstall: {
				Steps: []api.TestStep{{
					LiteralTestStep: &api.LiteralTestStep{
						As:       "ipi-lease",
						From:     "installer",
						Commands: "lease",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}, {
					LiteralTestStep: &api.LiteralTestStep{
						As:       "ipi-setup",
						From:     "installer",
						Commands: "openshift-cluster install",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
			},
		},
		expectedRes: api.MultiStageTestConfigurationLiteral{
			ClusterProfile: api.ClusterProfileAWS,
			Pre: []api.LiteralTestStep{{
				As:       "ipi-lease",
				From:     "installer",
				Commands: "lease",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}, {
				As:       "ipi-setup",
				From:     "installer",
				Commands: "openshift-cluster install",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}, {
				As:       "enable-fips",
				From:     "fips-enabler",
				Commands: "enable_fips",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				}},
			},
		},
	}, {
		name: "Test with duplicate names after unrolling chains",
		config: api.MultiStageTestConfiguration{
			ClusterProfile: api.ClusterProfileAWS,
			Pre: []api.TestStep{{
				Chain: &nestedChains,
			}},
		},
		chainMap: ChainByName{
			nestedChains: {
				Steps: []api.TestStep{{
					Chain: &chainInstall,
				}, {
					LiteralTestStep: &api.LiteralTestStep{
						As:       "ipi-setup",
						From:     "installer",
						Commands: "openshift-cluster install",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
			},
			chainInstall: {
				Steps: []api.TestStep{{
					LiteralTestStep: &api.LiteralTestStep{
						As:       "ipi-lease",
						From:     "installer",
						Commands: "lease",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}, {
					LiteralTestStep: &api.LiteralTestStep{
						As:       "ipi-setup",
						From:     "installer",
						Commands: "openshift-cluster install",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
			},
		},
		expectedRes: api.MultiStageTestConfigurationLiteral{},
		expectedErr: errors.New("test: nested-chains: duplicate name: ipi-setup"),
	}, {
		name: "Full AWS Workflow",
		config: api.MultiStageTestConfiguration{
			Workflow: &awsWorkflow,
		},
		chainMap: ChainByName{
			fipsPreChain: {
				Steps: []api.TestStep{{
					LiteralTestStep: &api.LiteralTestStep{
						As:       "ipi-install",
						From:     "installer",
						Commands: "openshift-cluster install",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}, {
					LiteralTestStep: &api.LiteralTestStep{
						As:       "enable-fips",
						From:     "fips-enabler",
						Commands: "enable_fips",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
			},
		},
		stepMap: ReferenceByName{
			teardownRef: {
				As:       "ipi-teardown",
				From:     "installer",
				Commands: "openshift-cluster destroy",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				}},
		},
		workflowMap: WorkflowByName{
			awsWorkflow: {
				ClusterProfile: api.ClusterProfileAWS,
				Pre: []api.TestStep{{
					Chain: &fipsPreChain,
				}},
				Test: []api.TestStep{{
					LiteralTestStep: &api.LiteralTestStep{
						As:       "e2e",
						From:     "my-image",
						Commands: "make custom-e2e",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
				Post: []api.TestStep{{
					Reference: &teardownRef,
				}},
			},
		},
		expectedRes: api.MultiStageTestConfigurationLiteral{
			ClusterProfile: api.ClusterProfileAWS,
			Pre: []api.LiteralTestStep{{
				As:       "ipi-install",
				From:     "installer",
				Commands: "openshift-cluster install",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				}}, {
				As:       "enable-fips",
				From:     "fips-enabler",
				Commands: "enable_fips",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				}},
			},
			Test: []api.LiteralTestStep{{
				As:       "e2e",
				From:     "my-image",
				Commands: "make custom-e2e",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
			Post: []api.LiteralTestStep{{
				As:       "ipi-teardown",
				From:     "installer",
				Commands: "openshift-cluster destroy",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
		},
	}, {
		name: "Workflow with Test and ClusterProfile overridden",
		config: api.MultiStageTestConfiguration{
			Workflow:       &awsWorkflow,
			ClusterProfile: api.ClusterProfileAzure4,
			Test: []api.TestStep{{
				LiteralTestStep: &api.LiteralTestStep{
					As:       "custom-e2e",
					From:     "test-image",
					Commands: "make custom-e2e-2",
					Resources: api.ResourceRequirements{
						Requests: api.ResourceList{"cpu": "1000m"},
						Limits:   api.ResourceList{"memory": "2Gi"},
					}},
			}},
		},
		workflowMap: WorkflowByName{
			awsWorkflow: {
				ClusterProfile: api.ClusterProfileAWS,
				Pre: []api.TestStep{{
					LiteralTestStep: &api.LiteralTestStep{
						As:       "ipi-install",
						From:     "installer",
						Commands: "openshift-cluster install",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
				Test: []api.TestStep{{
					LiteralTestStep: &api.LiteralTestStep{
						As:       "e2e",
						From:     "my-image",
						Commands: "make custom-e2e",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
				Post: []api.TestStep{{
					LiteralTestStep: &api.LiteralTestStep{
						As:       "ipi-teardown",
						From:     "installer",
						Commands: "openshift-cluster destroy",
						Resources: api.ResourceRequirements{
							Requests: api.ResourceList{"cpu": "1000m"},
							Limits:   api.ResourceList{"memory": "2Gi"},
						}},
				}},
			},
		},
		expectedRes: api.MultiStageTestConfigurationLiteral{
			ClusterProfile: api.ClusterProfileAzure4,
			Pre: []api.LiteralTestStep{{
				As:       "ipi-install",
				From:     "installer",
				Commands: "openshift-cluster install",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
			Test: []api.LiteralTestStep{{
				As:       "custom-e2e",
				From:     "test-image",
				Commands: "make custom-e2e-2",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
			Post: []api.LiteralTestStep{{
				As:       "ipi-teardown",
				From:     "installer",
				Commands: "openshift-cluster destroy",
				Resources: api.ResourceRequirements{
					Requests: api.ResourceList{"cpu": "1000m"},
					Limits:   api.ResourceList{"memory": "2Gi"},
				},
			}},
		},
	}} {
		t.Run(testCase.name, func(t *testing.T) {
			ret, err := NewResolver(testCase.stepMap, testCase.chainMap, testCase.workflowMap).Resolve("test", testCase.config)
			if testCase.expectedErr == nil {
				if err != nil {
					t.Fatalf("%s: expected no error but got: %s", testCase.name, err)
				}
			} else {
				if err == nil {
					t.Fatalf("%s: expected error but got none", testCase.name)
				}
				if testCase.expectedErr.Error() != err.Error() {
					t.Fatalf("%s: got incorrect error: %s", testCase.name, diff.ObjectReflectDiff(testCase.expectedErr, err))
				}
			}
			if !reflect.DeepEqual(ret, testCase.expectedRes) {
				t.Errorf("%s: got incorrect output: %s", testCase.name, diff.ObjectReflectDiff(ret, testCase.expectedRes))
			}
		})
	}
}
