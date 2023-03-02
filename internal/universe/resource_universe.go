package universe

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	client "github.com/yugabyte/platform-go-client"
	"github.com/yugabyte/terraform-provider-yugabyte-platform/internal/api"
	"github.com/yugabyte/terraform-provider-yugabyte-platform/internal/utils"
)

func ResourceUniverse() *schema.Resource {
	return &schema.Resource{
		Description: "Universe Resource",

		CreateContext: resourceUniverseCreate,
		ReadContext:   resourceUniverseRead,
		UpdateContext: resourceUniverseUpdate,
		DeleteContext: resourceUniverseDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Update: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(30 * time.Minute),
		},

		CustomizeDiff: resourceUniverseDiff(),
		Schema: map[string]*schema.Schema{
			// Universe Delete Options
			"delete_options": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delete_certs": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag indicating whether the certificates should be deleted with the universe",
						},
						"delete_backups": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "Flag indicating whether the backups should be deleted with the universe",
						},
						"force_delete": {
							Type:        schema.TypeBool,
							Optional:    true,
							Description: "", // TODO: document
						},
					},
				},
			},

			// Universe Fields
			"client_root_ca": {
				Type:     schema.TypeString,
				Optional: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// When TLS is enabled and this field is not set in the config file, a new root
					// certificate is created and this is populated. Subsequent runs will throw a
					// diff since this field is empty in the config file. This is to ignore the
					// difference in that case
					if len(old) > 0 && new == "" {
						return true
					}
					return false
				},
				Description: "", // TODO: document
			},
			"clusters": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Cluster UUID",
						},
						"cluster_type": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "Cluster Type. Permitted values: PRIMARY, ASYNC",
						},
						"user_intent": {
							Type:        schema.TypeList,
							MaxItems:    1,
							Required:    true,
							Elem:        userIntentSchema(),
							Description: "Configuration values used in universe creation. Only these values can be updated.",
						},
						"cloud_list": {
							Type:        schema.TypeList,
							Optional:    true,
							Computed:    true,
							Elem:        cloudListSchema(),
							Description: "Cloud, region, and zone placement information for the universe",
						},
					},
				},
			},
			"communication_ports": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"master_http_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"master_rpc_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"node_exporter_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"redis_server_http_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"redis_server_rpc_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"tserver_http_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"tserver_rpc_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"yql_server_http_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"yql_server_rpc_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"ysql_server_http_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"ysql_server_rpc_port": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func cloudListSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"uuid": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "", // TODO: document
			},
			"code": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "", // TODO: document
			},
			"region_list": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"uuid": {
							Type:        schema.TypeString,
							Computed:    true,
							Optional:    true,
							Description: "Region UUID",
						},
						"code": {
							Type:        schema.TypeString,
							Optional:    true,
							Computed:    true,
							Description: "", // TODO: document
						},
						"az_list": {
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"uuid": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "Zone UUID",
									},
									"is_affinitized": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "", // TODO: document
									},
									"name": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "Zone name",
									},
									"num_nodes": {
										Type:        schema.TypeInt,
										Optional:    true,
										Computed:    true,
										Description: "Number of nodes in this zone",
									},
									"replication_factor": {
										Type:        schema.TypeInt,
										Optional:    true,
										Computed:    true,
										Description: "Replication factor in this zone",
									},
									"secondary_subnet": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "", // TODO: document
									},
									"subnet": {
										Type:        schema.TypeString,
										Optional:    true,
										Computed:    true,
										Description: "", // TODO: document
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func userIntentSchema() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"assign_static_ip": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Flag indicating whether a static IP should be assigned",
			},
			"aws_arn_string": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"enable_exposing_service": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "", // TODO: document
			},
			"enable_ipv6": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"enable_ycql": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "", // TODO: document
			},
			"enable_ycql_auth": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"enable_ysql_auth": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"instance_tags": {
				Type:        schema.TypeMap,
				Elem:        &schema.Schema{Type: schema.TypeString},
				Optional:    true,
				Description: "", // TODO: document
			},
			"preferred_region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"use_host_name": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"use_systemd": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"ysql_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"ycql_password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"universe_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"provider_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"provider": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"region_list": {
				Type: schema.TypeList,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Optional:    true,
				Description: "", // TODO: document
			},
			"num_nodes": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Number of nodes for this universe",
			},
			"replication_factor": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Replicated factor for this universe",
			},
			"instance_type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"device_info": {
				Type:        schema.TypeList,
				MaxItems:    1,
				Required:    true,
				Description: "Configuration values associated with the machines used for this universe",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"disk_iops": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "", // TODO: document
						},
						"mount_points": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "", // TODO: document
						},
						"storage_class": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "", // TODO: document
						},
						"throughput": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "", // TODO: document
						},
						"num_volumes": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "", // TODO: document
						},
						"volume_size": {
							Type:        schema.TypeInt,
							Optional:    true,
							Description: "", // TODO: document
						},
						"storage_type": {
							Type:        schema.TypeString,
							Optional:    true,
							Description: "", // TODO: document
						},
					},
				},
			},
			"assign_public_ip": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"use_time_sync": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"enable_ysql": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "", // TODO: document
			},
			"enable_yedis": {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "", // TODO: document
			},
			"enable_node_to_node_encrypt": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"enable_client_to_node_encrypt": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"enable_volume_encryption": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "", // TODO: document
			},
			"yb_software_version": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"access_key_code": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"tserver_gflags": {
				Type:        schema.TypeMap,
				Elem:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
			"master_gflags": {
				Type:        schema.TypeMap,
				Elem:        schema.TypeString,
				Optional:    true,
				Description: "", // TODO: document
			},
		},
	}
}
func getClutserByType(clusters []client.Cluster, clusterType string) (client.Cluster, bool) {

	for _, v := range clusters {
		if v.ClusterType == clusterType {
			return v, true
		}
	}
	return client.Cluster{}, false
}
func resourceUniverseDiff() schema.CustomizeDiffFunc {
	return customdiff.All(
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// if not a new universe, prevent adding read replicas
			newClusterSet := buildClusters(new.([]interface{}))
			if len(old.([]interface{})) != 0 {
				oldClusterSet := buildClusters(old.([]interface{}))
				if len(oldClusterSet) < len(newClusterSet) {
					return errors.New("Cannot add Read Replica to existing universe")
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// if not a new universe, prevent systemD disablement
			newClusterSet := buildClusters(new.([]interface{}))
			if len(old.([]interface{})) != 0 {
				oldClusterSet := buildClusters(old.([]interface{}))
				oldPrimaryCluster, isPresent := getClutserByType(oldClusterSet, "PRIMARY")
				if isPresent {
					newPrimaryCluster, isNewPresent := getClutserByType(newClusterSet, "PRIMARY")
					if isNewPresent {
						if (oldPrimaryCluster.UserIntent.UseSystemd != nil) &&
							(*oldPrimaryCluster.UserIntent.UseSystemd == true &&
								((newPrimaryCluster.UserIntent.UseSystemd == nil) ||
									(*newPrimaryCluster.UserIntent.UseSystemd == false))) {
							return errors.New("Cannot disable SystemD")
						}
					}
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// if not a new universe, prevent decrease in volume size in primary
			newClusterSet := buildClusters(new.([]interface{}))
			if len(old.([]interface{})) != 0 {
				oldClusterSet := buildClusters(old.([]interface{}))
				oldPrimaryCluster, isPresent := getClutserByType(oldClusterSet, "PRIMARY")
				if isPresent {
					newPrimaryCluster, isNewPresent := getClutserByType(newClusterSet, "PRIMARY")
					if isNewPresent {
						if *oldPrimaryCluster.UserIntent.DeviceInfo.VolumeSize >
							*newPrimaryCluster.UserIntent.DeviceInfo.VolumeSize {
							return errors.New("Cannot decrease Volume Size of nodes in " +
								"Primary Cluster")
						}
					}
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// if not a new universe, prevent change in number of nodes if instance type hasn't
			// change in Primary
			newClusterSet := buildClusters(new.([]interface{}))
			if len(old.([]interface{})) != 0 {
				oldClusterSet := buildClusters(old.([]interface{}))
				oldPrimaryCluster, isPresent := getClutserByType(oldClusterSet, "PRIMARY")
				if isPresent {
					newPrimaryCluster, isNewPresent := getClutserByType(newClusterSet, "PRIMARY")
					if isNewPresent {
						if (*oldPrimaryCluster.UserIntent.InstanceType ==
							*newPrimaryCluster.UserIntent.InstanceType) &&
							(*oldPrimaryCluster.UserIntent.DeviceInfo.NumVolumes !=
								*newPrimaryCluster.UserIntent.DeviceInfo.NumVolumes) {
							return errors.New("Cannot change number of volumes per node " +
								"without change in instance type in Primary Cluster")
						}
					}
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// if not a new universe, prevent decrease in volume size in read replica
			newClusterSet := buildClusters(new.([]interface{}))
			if len(old.([]interface{})) != 0 {
				oldClusterSet := buildClusters(old.([]interface{}))
				oldPrimaryCluster, isPresent := getClutserByType(oldClusterSet, "ASYNC")
				if isPresent {
					newPrimaryCluster, isNewPresent := getClutserByType(newClusterSet, "ASYNC")
					if isNewPresent {
						if *oldPrimaryCluster.UserIntent.DeviceInfo.VolumeSize >
							*newPrimaryCluster.UserIntent.DeviceInfo.VolumeSize {
							return errors.New("Cannot decrease Volume Size of nodes in " +
								"Read Replica Cluster")
						}
					}
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// if not a new universe, prevent change in number of nodes if instance type hasn't
			// change in Read Replica
			newClusterSet := buildClusters(new.([]interface{}))
			if len(old.([]interface{})) != 0 {
				oldClusterSet := buildClusters(old.([]interface{}))
				oldPrimaryCluster, isPresent := getClutserByType(oldClusterSet, "ASYNC")
				if isPresent {
					newPrimaryCluster, isNewPresent := getClutserByType(newClusterSet, "ASYNC")
					if isNewPresent {
						if (*oldPrimaryCluster.UserIntent.InstanceType ==
							*newPrimaryCluster.UserIntent.InstanceType) &&
							((*oldPrimaryCluster.UserIntent.DeviceInfo.NumVolumes !=
								*newPrimaryCluster.UserIntent.DeviceInfo.NumVolumes) ||
								(*oldPrimaryCluster.UserIntent.DeviceInfo.VolumeSize !=
									*newPrimaryCluster.UserIntent.DeviceInfo.VolumeSize)) {
							return errors.New("Cannot change number of volumes or volume size " +
								"per node without change in instance type in Read Replica Cluster")
						}
					}
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// check if universe name of the clusters are the same
			newClusterSet := buildClusters(new.([]interface{}))
			newPrimary, isPresent := getClutserByType(newClusterSet, "PRIMARY")
			newReadOnly, isRRPresnt := getClutserByType(newClusterSet, "ASYNC")
			if isPresent && isRRPresnt {
				if newPrimary.UserIntent.UniverseName == nil {
					return errors.New("Universe name cannot be empty")
				}
				if newReadOnly.UserIntent.UniverseName == nil {
					return errors.New("Universe name cannot be empty")
				}
				if *newPrimary.UserIntent.UniverseName != *newReadOnly.UserIntent.UniverseName {
					return errors.New("Cannot have different universe names for Primary " +
						"and Read Only clusters")
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// check if software version of the clusters are the same
			newClusterSet := buildClusters(new.([]interface{}))
			newPrimary, isPresent := getClutserByType(newClusterSet, "PRIMARY")
			newReadOnly, isRRPresnt := getClutserByType(newClusterSet, "ASYNC")
			if (len(old.([]interface{})) != 0) {
				if isPresent && isRRPresnt {
					if (newPrimary.UserIntent.YbSoftwareVersion != nil) &&
						(newReadOnly.UserIntent.YbSoftwareVersion != nil) &&
						(*newPrimary.UserIntent.YbSoftwareVersion !=
							*newReadOnly.UserIntent.YbSoftwareVersion) {
						return errors.New("Cannot have different software versions for Primary " +
							"and Read Only clusters")
					}
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// check if systemD setting of the clusters are the same
			newClusterSet := buildClusters(new.([]interface{}))
			newPrimary, isPresent := getClutserByType(newClusterSet, "PRIMARY")
			newReadOnly, isRRPresnt := getClutserByType(newClusterSet, "ASYNC")
			if isPresent && isRRPresnt {
				if (newPrimary.UserIntent.UseSystemd != nil) &&
					(newReadOnly.UserIntent.UseSystemd != nil) &&
					(*newPrimary.UserIntent.UseSystemd != *newReadOnly.UserIntent.UseSystemd) {
					return errors.New("Cannot have different systemD settings for Primary " +
						"and Read Only clusters")
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// check if Gflags setting of the clusters are the same
			newClusterSet := buildClusters(new.([]interface{}))
			newPrimary, isPresent := getClutserByType(newClusterSet, "PRIMARY")
			newReadOnly, isRRPresnt := getClutserByType(newClusterSet, "ASYNC")
			if isPresent && isRRPresnt {
				if (newPrimary.UserIntent.MasterGFlags != nil) &&
					(newReadOnly.UserIntent.MasterGFlags != nil) &&
					(newPrimary.UserIntent.TserverGFlags != nil) &&
					(newReadOnly.UserIntent.TserverGFlags != nil) &&
					(!reflect.DeepEqual(*newPrimary.UserIntent.MasterGFlags,
						*newReadOnly.UserIntent.MasterGFlags) ||
					!reflect.DeepEqual(*newPrimary.UserIntent.TserverGFlags,
						*newReadOnly.UserIntent.TserverGFlags)) {
					return errors.New("Cannot have different Gflags settings for Primary " +
						"and Read Only clusters")
				}
			}
			return nil
		}),
		customdiff.ValidateChange("clusters", func(ctx context.Context, old, new, m interface{}) error {
			// check if TLS setting of the clusters are the same
			newClusterSet := buildClusters(new.([]interface{}))
			newPrimary, isPresent := getClutserByType(newClusterSet, "PRIMARY")
			newReadOnly, isRRPresnt := getClutserByType(newClusterSet, "ASYNC")
			if isPresent && isRRPresnt {
				if (newPrimary.UserIntent.EnableClientToNodeEncrypt != nil) &&
					(newReadOnly.UserIntent.EnableClientToNodeEncrypt != nil) &&
					(newPrimary.UserIntent.EnableNodeToNodeEncrypt != nil) &&
					(newReadOnly.UserIntent.EnableNodeToNodeEncrypt != nil) &&
					(*newPrimary.UserIntent.EnableClientToNodeEncrypt !=
						*newReadOnly.UserIntent.EnableClientToNodeEncrypt ||
					*newPrimary.UserIntent.EnableNodeToNodeEncrypt !=
						*newReadOnly.UserIntent.EnableNodeToNodeEncrypt) {
					return errors.New("Cannot have different TLS settings for Primary " +
						"and Read Only clusters")
				}
			}
			return nil
		}),
	)
}
func resourceUniverseCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*api.ApiClient).YugawareClient
	cUUID := meta.(*api.ApiClient).CustomerId

	req := buildUniverse(d)
	r, _, err := c.UniverseClusterMutationsApi.CreateAllClusters(ctx, cUUID).UniverseConfigureTaskParams(req).Execute()
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*r.ResourceUUID)
	tflog.Debug(ctx, fmt.Sprintf("Waiting for universe %s to be active", d.Id()))
	err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutCreate))
	if err != nil {
		return diag.FromErr(err)
	}
	return resourceUniverseRead(ctx, d, meta)
}

func resourceUniverseRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	c := meta.(*api.ApiClient).YugawareClient
	cUUID := meta.(*api.ApiClient).CustomerId

	r, _, err := c.UniverseManagementApi.GetUniverse(ctx, cUUID, d.Id()).Execute()
	if err != nil {
		return diag.FromErr(err)
	}

	u := r.UniverseDetails
	if err = d.Set("client_root_ca", u.ClientRootCA); err != nil {
		return diag.FromErr(err)
	}
	if err = d.Set("clusters", flattenClusters(u.Clusters)); err != nil {
		return diag.FromErr(err)
	}
	if err = d.Set("communication_ports", flattenCommunicationPorts(u.CommunicationPorts)); err != nil {
		return diag.FromErr(err)
	}
	return diags
}

func editUniverseParameters(ctx context.Context, old_user_intent client.UserIntent, new_user_intent client.UserIntent) (bool, client.UserIntent) {
	if !reflect.DeepEqual(*old_user_intent.InstanceTags, *new_user_intent.InstanceTags) ||
		!reflect.DeepEqual(*old_user_intent.RegionList, new_user_intent.RegionList) ||
		*old_user_intent.NumNodes != *new_user_intent.NumNodes ||
		*old_user_intent.InstanceType != *new_user_intent.InstanceType ||
		*old_user_intent.DeviceInfo.NumVolumes != *new_user_intent.DeviceInfo.NumVolumes ||
		*old_user_intent.DeviceInfo.VolumeSize != *new_user_intent.DeviceInfo.VolumeSize {
		edit_num_volume := true
		edit_volume_size := true // this is only for RR cluster, primary cluster resize is handled
		// by resize node task
		num_volumes := *old_user_intent.DeviceInfo.NumVolumes
		volume_size := *old_user_intent.DeviceInfo.VolumeSize
		if (*old_user_intent.DeviceInfo.NumVolumes != *new_user_intent.DeviceInfo.NumVolumes) &&
			(*old_user_intent.InstanceType == *new_user_intent.InstanceType) {
			tflog.Error(ctx, "Cannot edit Number of Volumes per instance without an edit to"+
				" Instance Type, Ignoring Change")
			edit_num_volume = false
		}
		if (*old_user_intent.DeviceInfo.VolumeSize != *new_user_intent.DeviceInfo.VolumeSize) &&
			(*old_user_intent.InstanceType == *new_user_intent.InstanceType) {
			tflog.Error(ctx, "Cannot edit Volume size per instance without an edit to Instance "+
				"Type, Ignoring Change for ReadOnly Cluster")
			tflog.Info(ctx, "Above error is not for Primary Cluster. Node resize applied through" +
				"a separate task")
			edit_volume_size = false
		} else if *old_user_intent.DeviceInfo.VolumeSize > *new_user_intent.DeviceInfo.VolumeSize {
			tflog.Error(ctx, "Cannot decrease volume size per instance, Ignoring Change")
			edit_volume_size = false
		}
		old_user_intent = new_user_intent
		if !edit_num_volume {
			old_user_intent.DeviceInfo.NumVolumes = &num_volumes
		}
		if !edit_volume_size {
			old_user_intent.DeviceInfo.VolumeSize = &volume_size
		}
		return true, old_user_intent
	}
	return false, old_user_intent

}

func resourceUniverseUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	// Only updates user intent for each cluster
	// cloud Info can have changes in zones
	c := meta.(*api.ApiClient).YugawareClient
	cUUID := meta.(*api.ApiClient).CustomerId

	if d.HasChange("clusters") {
		clusters := d.Get("clusters").([]interface{})
		updateUni, _, err := c.UniverseManagementApi.GetUniverse(ctx, cUUID, d.Id()).Execute()
		if err != nil {
			return diag.FromErr(errors.New(fmt.Sprintf("Unable to find universe %s", d.Id())))
		}
		newUni := buildUniverse(d)

		if len(clusters) > 2 {
			tflog.Error(ctx, "Cannot have more than 1 Read only cluster")
		} else {
			if len(updateUni.UniverseDetails.Clusters) < len(clusters) {
				tflog.Error(ctx, "Currently not supporting adding Read Replicas after universe creation")
			} else if len(updateUni.UniverseDetails.Clusters) > len(clusters) {
				var clusterUuid string
				for _, v := range updateUni.UniverseDetails.Clusters {
					if v.ClusterType == "ASYNC" {
						clusterUuid = *v.Uuid
					}
				}

				r, _, err := c.UniverseClusterMutationsApi.DeleteReadonlyCluster(ctx, cUUID, d.Id(), clusterUuid).
					IsForceDelete(d.Get("delete_options.0.force_delete").(bool)).Execute()
				if err != nil {
					return diag.FromErr(err)
				}
				tflog.Info(ctx, "DeleteReadOnlyCluster task is executing")
				err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutUpdate))
				if err != nil {
					return diag.FromErr(err)
				}
			}
		}
		for i, v := range clusters {
			if !d.HasChange(fmt.Sprintf("clusters.%d", i)) {
				continue
			}
			cluster := v.(map[string]interface{})

			old_user_intent := updateUni.UniverseDetails.Clusters[i].UserIntent
			new_user_intent := newUni.Clusters[i].UserIntent
			if cluster["cluster_type"] == "PRIMARY" {

				//Software Upgrade
				if *old_user_intent.YbSoftwareVersion != *new_user_intent.YbSoftwareVersion {
					updateUni.UniverseDetails.Clusters[i].UserIntent = new_user_intent
					req := client.SoftwareUpgradeParams{
						YbSoftwareVersion: *new_user_intent.YbSoftwareVersion,
						Clusters:          updateUni.UniverseDetails.Clusters,
						UpgradeOption:     "Rolling",
					}
					r, _, err := c.UniverseUpgradesManagementApi.UpgradeSoftware(ctx, cUUID, d.Id()).SoftwareUpgradeParams(req).Execute()
					if err != nil {
						return diag.FromErr(err)
					}
					tflog.Info(ctx, "UpgradeSoftware task is executing")
					err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutUpdate))
					if err != nil {
						return diag.FromErr(err)
					}
				}

				updateUni, _, err = c.UniverseManagementApi.GetUniverse(ctx, cUUID, d.Id()).Execute()
				if err != nil {
					return diag.FromErr(errors.New(fmt.Sprintf("Unable to find universe %s", d.Id())))
				}
				old_user_intent = updateUni.UniverseDetails.Clusters[i].UserIntent

				//GFlag Update
				if !reflect.DeepEqual(*old_user_intent.MasterGFlags, *new_user_intent.MasterGFlags) ||
					!reflect.DeepEqual(*old_user_intent.TserverGFlags, *new_user_intent.TserverGFlags) {
					updateUni.UniverseDetails.Clusters[i].UserIntent = new_user_intent
					req := client.GFlagsUpgradeParams{
						MasterGFlags:  *new_user_intent.MasterGFlags,
						TserverGFlags: *new_user_intent.TserverGFlags,
						Clusters:      updateUni.UniverseDetails.Clusters,
						UpgradeOption: "Rolling",
					}
					r, _, err := c.UniverseUpgradesManagementApi.UpgradeGFlags(ctx, cUUID, d.Id()).GflagsUpgradeParams(req).Execute()
					if err != nil {
						return diag.FromErr(err)
					}
					tflog.Info(ctx, "UpgradeGFlags task is executing")
					err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutUpdate))
					if err != nil {
						return diag.FromErr(err)
					}
				}

				updateUni, _, err = c.UniverseManagementApi.GetUniverse(ctx, cUUID, d.Id()).Execute()
				if err != nil {
					return diag.FromErr(errors.New(fmt.Sprintf("Unable to find universe %s", d.Id())))
				}
				old_user_intent = updateUni.UniverseDetails.Clusters[i].UserIntent

				//TLS Toggle
				if *old_user_intent.EnableClientToNodeEncrypt != *new_user_intent.EnableClientToNodeEncrypt ||
					*old_user_intent.EnableNodeToNodeEncrypt != *new_user_intent.EnableNodeToNodeEncrypt {
					updateUni.UniverseDetails.Clusters[i].UserIntent = new_user_intent
					req := client.TlsToggleParams{
						EnableClientToNodeEncrypt: *new_user_intent.EnableClientToNodeEncrypt,
						EnableNodeToNodeEncrypt:   *new_user_intent.EnableNodeToNodeEncrypt,
						Clusters:                  updateUni.UniverseDetails.Clusters,
						UpgradeOption:             "Non-Rolling",
					}
					r, _, err := c.UniverseUpgradesManagementApi.UpgradeTls(ctx, cUUID, d.Id()).TlsToggleParams(req).Execute()
					if err != nil {
						return diag.FromErr(err)
					}
					tflog.Info(ctx, "UpgradeTLS task is executing")
					err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutUpdate))
					if err != nil {
						return diag.FromErr(err)
					}
				}

				updateUni, _, err = c.UniverseManagementApi.GetUniverse(ctx, cUUID, d.Id()).Execute()
				if err != nil {
					return diag.FromErr(errors.New(fmt.Sprintf("Unable to find universe %s", d.Id())))
				}
				old_user_intent = updateUni.UniverseDetails.Clusters[i].UserIntent

				//SystemD upgrade
				if (new_user_intent.UseSystemd != nil) && (*old_user_intent.UseSystemd != *new_user_intent.UseSystemd) &&
					(*old_user_intent.UseSystemd == false) {
					updateUni.UniverseDetails.Clusters[i].UserIntent = new_user_intent
					req := client.SystemdUpgradeParams{

						Clusters:      updateUni.UniverseDetails.Clusters,
						UpgradeOption: "Rolling",
					}
					r, _, err := c.UniverseUpgradesManagementApi.UpgradeSystemd(ctx, cUUID, d.Id()).SystemdUpgradeParams(req).Execute()
					if err != nil {
						return diag.FromErr(err)
					}
					tflog.Info(ctx, "UpgradeSystemd task is executing")
					err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutUpdate))
					if err != nil {
						return diag.FromErr(err)
					}
				} else if *old_user_intent.UseSystemd == true &&
					new_user_intent.UseSystemd == nil || *new_user_intent.UseSystemd == false {
					tflog.Error(ctx, "Cannot disable Systemd")
				}

				updateUni, _, err = c.UniverseManagementApi.GetUniverse(ctx, cUUID, d.Id()).Execute()
				if err != nil {
					return diag.FromErr(errors.New(fmt.Sprintf("Unable to find universe %s", d.Id())))
				}
				old_user_intent = updateUni.UniverseDetails.Clusters[i].UserIntent

				// Resize Nodes
				// Call separate task only when instance type is same, else will be handled in
				// UpdatePrimaryCluster
				if (*old_user_intent.InstanceType == *new_user_intent.InstanceType) &&
					(*old_user_intent.DeviceInfo.VolumeSize != *new_user_intent.DeviceInfo.VolumeSize) {
					if *old_user_intent.DeviceInfo.VolumeSize < *new_user_intent.DeviceInfo.VolumeSize {
						//Only volume size should be changed to do smart resize, other changes handled in UpgradeCluster
						updateUni.UniverseDetails.Clusters[i].UserIntent.DeviceInfo.VolumeSize = new_user_intent.DeviceInfo.VolumeSize
						req := client.ResizeNodeParams{
							UpgradeOption:  "Rolling",
							Clusters:       updateUni.UniverseDetails.Clusters,
							NodeDetailsSet: buildNodeDetailsRespArrayToNodeDetailsArray(updateUni.UniverseDetails.NodeDetailsSet),
						}
						r, _, err := c.UniverseUpgradesManagementApi.ResizeNode(ctx, cUUID, d.Id()).ResizeNodeParams(req).Execute()
						if err != nil {
							return diag.FromErr(err)
						}
						tflog.Info(ctx, "ResizeNode task is executing")
						err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutUpdate))
						if err != nil {
							return diag.FromErr(err)
						}
					} else {
						tflog.Error(ctx, "Volume Size cannot be decreased")
					}
				}

				updateUni, _, err = c.UniverseManagementApi.GetUniverse(ctx, cUUID, d.Id()).Execute()
				if err != nil {
					return diag.FromErr(errors.New(fmt.Sprintf("Unable to find universe %s", d.Id())))
				}
				old_user_intent = updateUni.UniverseDetails.Clusters[i].UserIntent

				// Num of nodes, Instance Type, Num of Volumes, Volume Size, User Tags changes
				var edit_allowed, edit_zone_allowed bool
				edit_allowed, updateUni.UniverseDetails.Clusters[i].UserIntent = editUniverseParameters(ctx, old_user_intent, new_user_intent)
				if edit_allowed || edit_zone_allowed {
					req := client.UniverseConfigureTaskParams{
						UniverseUUID:   utils.GetStringPointer(d.Id()),
						Clusters:       updateUni.UniverseDetails.Clusters,
						NodeDetailsSet: buildNodeDetailsRespArrayToNodeDetailsArray(updateUni.UniverseDetails.NodeDetailsSet),
					}
					r, _, err := c.UniverseClusterMutationsApi.UpdatePrimaryCluster(ctx, cUUID, d.Id()).UniverseConfigureTaskParams(req).Execute()
					if err != nil {
						return diag.FromErr(err)
					}
					tflog.Info(ctx, "UpdatePrimaryCluster task is executing")
					err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutUpdate))
					if err != nil {
						return diag.FromErr(err)
					}
				}

			} else {

				//Ignore Software, GFlags, SystemD, TLS Upgrade changes to Read-Only Cluster
				updateUni, _, err := c.UniverseManagementApi.GetUniverse(ctx, cUUID, d.Id()).Execute()
				if err != nil {
					return diag.FromErr(errors.New(fmt.Sprintf("Unable to find universe %s", d.Id())))
				}
				old_user_intent := updateUni.UniverseDetails.Clusters[i].UserIntent
				if *old_user_intent.YbSoftwareVersion != *new_user_intent.YbSoftwareVersion {
					tflog.Info(ctx, "Software Upgrade is applied only via change in Primary Cluster User Intent, ignoring")
				}
				if !reflect.DeepEqual(*old_user_intent.MasterGFlags, *new_user_intent.MasterGFlags) ||
					!reflect.DeepEqual(*old_user_intent.TserverGFlags, *new_user_intent.TserverGFlags) {
					tflog.Info(ctx, "GFlags Upgrade is applied only via change in Primary Cluster User Intent, ignoring")
				}
				if (new_user_intent.UseSystemd != nil) && (*old_user_intent.UseSystemd != *new_user_intent.UseSystemd) {
					tflog.Info(ctx, "System Upgrade is applied only via change in Primary Cluster User Intent, ignoring")
				}
				if *old_user_intent.EnableClientToNodeEncrypt != *new_user_intent.EnableClientToNodeEncrypt ||
					*old_user_intent.EnableNodeToNodeEncrypt != *new_user_intent.EnableNodeToNodeEncrypt {
					tflog.Info(ctx, "TLS Toggle is applied only via change in Primary Cluster User Intent, ignoring")
				}

				// Num of nodes, Instance Type, Num of Volumes, Volume Size User Tags changes
				var edit_allowed bool
				edit_allowed, updateUni.UniverseDetails.Clusters[i].UserIntent = editUniverseParameters(ctx, old_user_intent, new_user_intent)
				if edit_allowed {
					req := client.UniverseConfigureTaskParams{
						UniverseUUID:   utils.GetStringPointer(d.Id()),
						Clusters:       updateUni.UniverseDetails.Clusters,
						NodeDetailsSet: buildNodeDetailsRespArrayToNodeDetailsArray(updateUni.UniverseDetails.NodeDetailsSet),
					}
					r, _, err := c.UniverseClusterMutationsApi.UpdateReadOnlyCluster(ctx, cUUID, d.Id()).UniverseConfigureTaskParams(req).Execute()
					if err != nil {
						return diag.FromErr(err)
					}
					tflog.Info(ctx, "UpdateReadOnlyCluster task is executing")
					err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutUpdate))
					if err != nil {
						return diag.FromErr(err)
					}
				}

				if (*old_user_intent.EnableClientToNodeEncrypt != *new_user_intent.EnableClientToNodeEncrypt) ||
					(*old_user_intent.EnableNodeToNodeEncrypt != *new_user_intent.EnableNodeToNodeEncrypt) {
					tflog.Info(ctx, "TLS Upgrade is applied only via change in Primary Cluster User Intent, ignoring")
				}
			}

		}
	}

	return resourceUniverseRead(ctx, d, meta)
}

func resourceUniverseDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	c := meta.(*api.ApiClient).YugawareClient
	cUUID := meta.(*api.ApiClient).CustomerId

	r, _, err := c.UniverseManagementApi.DeleteUniverse(ctx, cUUID, d.Id()).
		IsForceDelete(d.Get("delete_options.0.force_delete").(bool)).
		IsDeleteBackups(d.Get("delete_options.0.delete_backups").(bool)).
		IsDeleteAssociatedCerts(d.Get("delete_options.0.delete_certs").(bool)).
		Execute()
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, fmt.Sprintf("Waiting for universe %s to be deleted", d.Id()))
	err = utils.WaitForTask(ctx, *r.TaskUUID, cUUID, c, d.Timeout(schema.TimeoutDelete))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return diags
}
